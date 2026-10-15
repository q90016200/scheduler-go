package scheduler

import (
	"context"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/jhump/protoreflect/desc"
	"github.com/q90016200/scheduler-go/config"
	"github.com/q90016200/scheduler-go/internal/models"
	"github.com/q90016200/scheduler-go/internal/ssrf"
	"github.com/q90016200/scheduler-go/pkg/logger"
	"github.com/robfig/cron/v3"
	"google.golang.org/grpc"
	"gorm.io/gorm"
	"net/http"
	"sync"
	"time"
)

// runBatch 進行中的「立即執行」批次：id 對應 TaskExecutionLog.BatchID，cancel 用來中止整個批次
type runBatch struct {
	id     string
	cancel context.CancelFunc
}

// CronScheduler 全局的排程器實例
type CronScheduler struct {
	Cron             *cron.Cron
	Log              *logger.LogrusAdapter
	taskCronEntryIDs map[string]cron.EntryID // Task ID 對應其 cron EntryID 的映射
	mu               sync.Mutex              // 保證多協程安全
	db               *gorm.DB
	restyClient      *resty.Client
	runBatches       map[string]runBatch // Task ID 對應其進行中的「立即執行」重複批次，供取消與查詢是否仍在跑
	runBatchesMu     sync.Mutex          // 保證多協程安全，與 mu 分開避免不相關操作互相卡鎖
	batchWG          sync.WaitGroup      // 追蹤「立即執行」批次 goroutine，Stop 時限時等待收尾
	baseCtx          context.Context     // 所有批次共同的父 context，Stop 時一併取消尚未跑完的批次
	baseCancel       context.CancelFunc
	rpcMu            sync.Mutex                        // 保護下面三個 gRPC 快取欄位；不與 mu / runBatchesMu 共用
	rpcConns         map[string]*grpc.ClientConn       // endpoint 對應的 gRPC 連線快取
	rpcMethods       map[string]*desc.MethodDescriptor // endpoint+"|"+method 對應的 MethodDescriptor 快取
	rpcClosed        bool                              // Stop 之後為 true，不再建立新連線
}

func NewCronScheduler(db *gorm.DB) *CronScheduler {
	log := logger.Log
	cronLogger := &logger.LogrusAdapter{Logger: log}
	baseCtx, baseCancel := context.WithCancel(context.Background())
	return &CronScheduler{
		Cron:             cron.New(cron.WithChain(cron.Recover(cronLogger), cron.SkipIfStillRunning(cronLogger))),
		Log:              cronLogger,
		taskCronEntryIDs: make(map[string]cron.EntryID),
		db:               db,
		restyClient:      newSSRFGuardedRestyClient(),
		runBatches:       make(map[string]runBatch),
		baseCtx:          baseCtx,
		baseCancel:       baseCancel,
		rpcConns:         make(map[string]*grpc.ClientConn),
		rpcMethods:       make(map[string]*desc.MethodDescriptor),
	}
}

// newSSRFGuardedRestyClient 建立一個限制連線目標的 resty client，避免無 auth 的 http 任務
// 被當作 SSRF 跳板探測 loopback / link-local / metadata 位址（見 internal/ssrf.SafeDialContext）
func newSSRFGuardedRestyClient() *resty.Client {
	transport := &http.Transport{DialContext: ssrf.SafeDialContext}
	// 不在 client 上設 timeout：逾時由每次嘗試的 context（task.TimeoutSeconds）決定，
	// 否則 http.Client.Timeout 會先截斷超過 10 秒的任務
	return resty.New().SetTransport(transport)
}

// jobCount 持鎖回傳目前已註冊的 cron job 數量
func (cs *CronScheduler) jobCount() int {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return len(cs.taskCronEntryIDs)
}

// NextRunAts 一次回傳多個任務的下次執行時間（key 為 task id）；未掛上 cron job（未啟用、或註冊失敗）的任務不在結果內。
// robfig/cron 的 Entry(id) 內部是 Entries()（運行中要經 channel 向排程 loop 要快照）再線性掃描，
// 逐列呼叫會複製 N 次全部 entry，所以這裡持鎖只取 id→entryID 快照，鎖外只呼叫一次 Entries()。
func NextRunAts(cs *CronScheduler, ids []string) map[string]time.Time {
	cs.mu.Lock()
	entryIDs := make(map[string]cron.EntryID, len(ids))
	for _, id := range ids {
		if entryID, ok := cs.taskCronEntryIDs[id]; ok {
			entryIDs[id] = entryID
		}
	}
	cs.mu.Unlock()
	if len(entryIDs) == 0 {
		return nil
	}

	nexts := make(map[cron.EntryID]time.Time)
	for _, e := range cs.Cron.Entries() {
		nexts[e.ID] = e.Next
	}
	result := make(map[string]time.Time, len(entryIDs))
	for id, entryID := range entryIDs {
		if next := nexts[entryID]; !next.IsZero() {
			result[id] = next
		}
	}
	return result
}

// reconcile 每分鐘 poller 的本體：比對「DB 中 enabled 的任務」與「已註冊的 cron entry」，
// 對差集（enabled 但未註冊、已註冊但不在 enabled 清單）呼叫一次 SyncTasks（一次鎖、一條 id IN 查詢）。
// 已註冊 id 的快照在 cs.mu 內取、Pluck 在鎖外，兩者之間若有變動只會讓某筆被多／少同步一輪；
// 真正的狀態判斷仍由 SyncTasks 持鎖重讀 DB 決定，不信任這份快照，所以不會讓已停用的任務復活。
// SyncTasks 自己持鎖，因此必須在鎖外呼叫以免巢狀鎖。
func (cs *CronScheduler) reconcile() {
	cs.mu.Lock()
	registered := make(map[string]struct{}, len(cs.taskCronEntryIDs))
	for id := range cs.taskCronEntryIDs {
		registered[id] = struct{}{}
	}
	cs.mu.Unlock()

	var ids []string
	err := cs.db.Model(&models.Task{}).Where("status = ?", models.STATUS_ENABLED).
		Order("created_at ASC").
		Pluck("id", &ids).Error
	if err != nil {
		cs.Log.Error(err, "reconcile: pluck enabled task ids err")
		return
	}

	var diff []string
	for _, id := range ids {
		if _, ok := registered[id]; ok {
			delete(registered, id) // 兩邊都有；剩下的就是「已註冊但不在 enabled 清單」
		} else {
			diff = append(diff, id)
		}
	}
	for id := range registered {
		diff = append(diff, id)
	}
	if err := SyncTasks(cs, diff); err != nil {
		cs.Log.Error(err, "poller SyncTasks err")
	}
}

// StartScheduler 啟動排程器並註冊所有任務
func (cs *CronScheduler) StartScheduler() {
	// 建立每分鐘檢查任務狀態任務
	_, err := cs.Cron.AddFunc("* * * * *", func() {
		cs.Log.Info(fmt.Sprintf("[cron] Running cron scheduler jobs, registered=%d", cs.jobCount()))
		cs.reconcile()
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

	// 啟動前先對帳一次，讓 enabled 任務一啟動就註冊，不必等第一個整分鐘的 poller
	cs.reconcile()

	cs.Cron.Start()
}

// stopWaitTimeout StopCronScheduler 每個等待階段的上限；兩階段加 main 的 srv.Shutdown 最壞約 30 秒，
// 超過 docker 預設 10 秒的 stop 寬限，部署需加 --stop-timeout 30（見 README）
const stopWaitTimeout = 10 * time.Second

// waitOrLog 在 stopWaitTimeout 內等 wait 返回，逾時只記 log 就放行，不讓關機卡死
func (cs *CronScheduler) waitOrLog(what string, wait func()) {
	done := make(chan struct{})
	go func() {
		wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(stopWaitTimeout):
		cs.Log.Info(fmt.Sprintf("等待%s結束逾時（%s），繼續關機", what, stopWaitTimeout))
	}
}

// StopCronScheduler 停止排程器：先停止觸發新的 cron job，再取消 baseCtx 中止進行中的 cron job 與
// 「立即執行」批次（取消會一路傳進 resty／gRPC 呼叫與重試等待），之後限時等它們收尾（含寫完最後一筆紀錄），最後關 gRPC 連線。
// 順序不能反：等待必須在 baseCancel 之後，否則會白等滿逾時。
// ponytail: 未追蹤 notify.Dispatch 的通知 goroutine（屬 internal/notify），關機時進行中的通知可能被截斷；
// 要保證送出再讓 Dispatch 回傳 WaitGroup/等待函式。
func (cs *CronScheduler) StopCronScheduler() {
	ctx := cs.Cron.Stop()
	cs.Log.Info("排程器已停止觸發新任務")
	cs.baseCancel()
	// 屏障：與 RunTaskNow 在 runBatchesMu 內的 baseCtx 檢查＋batchWG.Add 互斥，確保 Add 都在 Wait 之前
	cs.runBatchesMu.Lock()
	cs.runBatchesMu.Unlock()
	cs.waitOrLog("進行中的 cron 任務", func() { <-ctx.Done() })
	cs.waitOrLog("進行中的立即執行批次", cs.batchWG.Wait)
	cs.closeRPCConns()
}
