package scheduler

import (
	"fmt"
	"github.com/q90016200/scheduler-go/internal/app"
	"github.com/q90016200/scheduler-go/internal/models"
	"github.com/q90016200/scheduler-go/pkg/logger"
	"github.com/robfig/cron/v3"
	"sync"
)

// CronScheduler 全局的排程器實例
type CronScheduler struct {
	Cron   *cron.Cron
	Log    *logger.LogrusAdapter
	jobIDs map[string]cron.EntryID // 任務名稱與 EntryID 的映射
	mu     sync.Mutex              // 保證多協程安全
}

func NewCronScheduler() *CronScheduler {
	log := logger.Logger
	cronLogger := &logger.LogrusAdapter{Logger: log}
	return &CronScheduler{
		Cron:   cron.New(cron.WithChain(cron.SkipIfStillRunning(cronLogger))),
		Log:    cronLogger,
		jobIDs: make(map[string]cron.EntryID),
	}
}

// StartScheduler 啟動排程器並註冊所有任務
func (cs *CronScheduler) StartScheduler(service *app.AppServices) {
	// 建立每分鐘檢查任務狀態任務
	_, err := cs.Cron.AddFunc("@every 1m", func() {
		tasks, err := service.TaskServices.ListTasks(map[string]interface{}{
			"status": models.TASK_STATUS_ACTIVE,
		}, "created_at", "ASC")
		if err != nil {
			errMsg := fmt.Sprintf("TaskServices.ListTasks err: %v", err)
			cs.Log.Error(nil, errMsg)
			return
		}
		for _, task := range tasks {
			_, exists := cs.jobIDs[task.ID]
			if task.Status == models.TASK_STATUS_ACTIVE {
				if !exists {
					cs.AddJob(task.ID, &task)
				}
			} else {
				if exists {

				}
			}
		}
	})
	if err != nil {
		logger.Logger.WithError(err).Error("Failed to scheduler cleanup task")
	}

	cs.Cron.Start()
}

// StopCronScheduler 停止排程器
func (cs *CronScheduler) StopCronScheduler() {
	if cs.Cron != nil {
		cs.Cron.Stop()
		logger.Logger.Info("排程器已停止")
	}
}
