package scheduler

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/q90016200/scheduler-go/config"
	"github.com/q90016200/scheduler-go/internal/models"
	"github.com/q90016200/scheduler-go/pkg/logger"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
	"sync"
	"time"
)

// CronScheduler 全局的排程器實例
type CronScheduler struct {
	Cron        *cron.Cron
	Log         *logger.LogrusAdapter
	jobIDs      map[string]cron.EntryID // 任務名稱與 EntryID 的映射
	mu          sync.Mutex              // 保證多協程安全
	db          *gorm.DB
	restyClient *resty.Client
}

func NewCronScheduler(db *gorm.DB) *CronScheduler {
	//log := logger.Init("../../logs/cron.log")
	log := logger.Log
	cronLogger := &logger.LogrusAdapter{Logger: log}
	return &CronScheduler{
		Cron:        cron.New(cron.WithChain(cron.SkipIfStillRunning(cronLogger))),
		Log:         cronLogger,
		jobIDs:      make(map[string]cron.EntryID),
		db:          db,
		restyClient: resty.New(),
	}
}

// StartScheduler 啟動排程器並註冊所有任務
func (cs *CronScheduler) StartScheduler() {
	// 建立每分鐘檢查任務狀態任務
	_, err := cs.Cron.AddFunc("* * * * *", func() {
		fmt.Println("[cron] Running cron scheduler jobs", cs.jobIDs)
		var tasks []models.Task
		err := cs.db.Where("status = ?", models.TASK_STATUS_ACTIVE).
			Order("created_at ASC").
			Preload("Group").
			Find(&tasks).Error
		if err != nil {
			errMsg := fmt.Sprintf("TaskServices.ListTasks err: %v", err)
			cs.Log.Error(nil, errMsg)
			return
		}
		for _, task := range tasks {
			_, exists := cs.jobIDs[task.ID]
			if task.Status == models.TASK_STATUS_ACTIVE {
				if !exists {
					err = AddJob(cs, task.ID, &task)
					if err != nil {
						cs.Log.Info("task ID: " + task.ID + ", AddJob err:" + err.Error())
					}
				}
			} else {
				if exists {
					err = RemoveJob(cs, task.ID)
					if err != nil {
						cs.Log.Info("task ID: " + task.ID + ", RemoveJob err:" + err.Error())
					}
				}
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

// StopCronScheduler 停止排程器
func (cs *CronScheduler) StopCronScheduler() {
	if cs.Cron != nil {
		cs.Cron.Stop()
		cs.Log.Info("排程器已停止")
	}
}
