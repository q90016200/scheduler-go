package scheduler

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/q90016200/scheduler-go/config"
	"github.com/q90016200/scheduler-go/internal/models"
	"github.com/q90016200/scheduler-go/internal/ssrf"
	"github.com/q90016200/scheduler-go/pkg/logger"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
	"net/http"
	"sync"
	"time"
)

// CronScheduler 全局的排程器實例
type CronScheduler struct {
	Cron             *cron.Cron
	Log              *logger.LogrusAdapter
	taskCronEntryIDs map[string]cron.EntryID // Task ID 對應其 cron EntryID 的映射
	mu               sync.Mutex              // 保證多協程安全
	db               *gorm.DB
	restyClient      *resty.Client
	runBatches       map[string]context.CancelFunc // Task ID 對應其進行中的「立即執行」重複批次，供取消用
	runBatchesMu     sync.Mutex                    // 保證多協程安全，與 mu 分開避免不相關操作互相卡鎖
	baseCtx          context.Context               // 所有批次共同的父 context，Stop 時一併取消尚未跑完的批次
	baseCancel       context.CancelFunc
}

func NewCronScheduler(db *gorm.DB) *CronScheduler {
	//log := logger.Init("../../logs/cron.log")
	log := logger.Log
	cronLogger := &logger.LogrusAdapter{Logger: log}
	baseCtx, baseCancel := context.WithCancel(context.Background())
	return &CronScheduler{
		Cron:             cron.New(cron.WithChain(cron.Recover(cronLogger), cron.SkipIfStillRunning(cronLogger))),
		Log:              cronLogger,
		taskCronEntryIDs: make(map[string]cron.EntryID),
		db:               db,
		restyClient:      newSSRFGuardedRestyClient(),
		runBatches:       make(map[string]context.CancelFunc),
		baseCtx:          baseCtx,
		baseCancel:       baseCancel,
	}
}

// newSSRFGuardedRestyClient 建立一個限制連線目標的 resty client，避免無 auth 的 http 任務
// 被當作 SSRF 跳板探測 loopback / link-local / metadata 位址（見 internal/ssrf.SafeDialContext）
func newSSRFGuardedRestyClient() *resty.Client {
	transport := &http.Transport{DialContext: ssrf.SafeDialContext}
	return resty.New().SetTimeout(10 * time.Second).SetTransport(transport)
}

// hasJob 回傳 taskID 是否已註冊 cron job，持鎖以避免與 Register/UnregisterTaskJob 的併發存取
func (cs *CronScheduler) hasJob(taskID string) bool {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	_, exists := cs.taskCronEntryIDs[taskID]
	return exists
}

// jobCount 持鎖回傳目前已註冊的 cron job 數量
func (cs *CronScheduler) jobCount() int {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return len(cs.taskCronEntryIDs)
}

// registerIfStillEnabled 供每分鐘 poller 使用：task 是否仍為 enabled 的判斷跟註冊動作
// 必須在同一次持鎖期間完成，否則會有 race——poller 先 SELECT 出一批 enabled 任務的快照，
// 若在它跑到某個 task 之前，該 task 被 disable/刪除且 UnregisterTaskJob 已經執行過，
// poller 若只憑舊快照直接註冊，就會把一個已經停用的任務重新排回 cron、永遠停不下來
// （poller 本身只會新增 job，從不移除）。
// ponytail: 這裡在持有 cs.mu 期間查了一次 DB，等於把鎖的臨界區擴大到含一次 DB round trip；
// 但只有「目前不在 taskCronEntryIDs 裡」的任務才會走到這個查詢，數量很少，可接受。
func (cs *CronScheduler) registerIfStillEnabled(taskID string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if _, exists := cs.taskCronEntryIDs[taskID]; exists {
		return
	}

	var fresh models.Task
	err := cs.db.Where("id = ? AND status = ?", taskID, models.STATUS_ENABLED).Preload("Group").First(&fresh).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			cs.Log.Error(err, "task ID: "+taskID+", poller 重新查詢任務狀態失敗")
		}
		return
	}

	if err := registerTaskJobLocked(cs, &fresh); err != nil {
		cs.Log.Error(err, "task ID: "+taskID+", RegisterTaskJob err")
	}
}

// NextRunAt 回傳任務下次執行時間；任務未掛上 cron job（未啟用、或註冊失敗）時 ok 為 false
func NextRunAt(cs *CronScheduler, taskID string) (time.Time, bool) {
	cs.mu.Lock()
	entryID, exists := cs.taskCronEntryIDs[taskID]
	cs.mu.Unlock()
	if !exists {
		return time.Time{}, false
	}
	next := cs.Cron.Entry(entryID).Next
	if next.IsZero() {
		return time.Time{}, false
	}
	return next, true
}

// StartScheduler 啟動排程器並註冊所有任務
func (cs *CronScheduler) StartScheduler() {
	// 建立每分鐘檢查任務狀態任務
	_, err := cs.Cron.AddFunc("* * * * *", func() {
		cs.Log.Info(fmt.Sprintf("[cron] Running cron scheduler jobs, registered=%d", cs.jobCount()))
		var tasks []models.Task
		err := cs.db.Where("status = ?", models.STATUS_ENABLED).
			Order("created_at ASC").
			Preload("Group").
			Find(&tasks).Error
		if err != nil {
			cs.Log.Error(err, "TaskServices.ListTasks err")
			return
		}
		for _, task := range tasks {
			// hasJob 只是快速篩選，真正的「是否仍 enabled」要在鎖內重查，見 registerIfStillEnabled 註解
			if !cs.hasJob(task.ID) {
				cs.registerIfStillEnabled(task.ID)
			}
		}
	})
	if err != nil {
		cs.Log.Error(err, "Failed to scheduler cleanup task")
	}

	// 每日清除超過保留天數的任務執行紀錄
	_, err = cs.Cron.AddFunc("0 3 * * *", func() {
		cutoff := time.Now().AddDate(0, 0, -config.AppConfig.ExecutionLogRetentionDays).Unix()
		result := cs.db.Where("created_at < ?", cutoff).Delete(&models.TaskExecutionLog{})
		if result.Error != nil {
			cs.Log.Error(result.Error, "清除過期執行紀錄失敗")
			return
		}
		cs.Log.Info(fmt.Sprintf("已清除 %d 筆超過 %d 天的執行紀錄", result.RowsAffected, config.AppConfig.ExecutionLogRetentionDays))
	})
	if err != nil {
		cs.Log.Error(err, "Failed to schedule execution log cleanup task")
	}

	cs.Cron.Start()
}

// StopCronScheduler 停止排程器，並取消所有進行中的「立即執行」重複批次（已發出的請求仍會跑完）
func (cs *CronScheduler) StopCronScheduler() {
	if cs.Cron != nil {
		cs.Cron.Stop()
		cs.Log.Info("排程器已停止")
	}
	if cs.baseCancel != nil {
		cs.baseCancel()
	}
}
