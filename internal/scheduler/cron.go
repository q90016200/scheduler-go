package scheduler

import (
	"fmt"
	"github.com/q90016200/scheduler-go/internal/app"
	"github.com/q90016200/scheduler-go/internal/models"
	"github.com/q90016200/scheduler-go/pkg/logger"
	"github.com/robfig/cron/v3"
)

// Scheduler 全局的排程器實例
var Scheduler *cron.Cron

// StartScheduler 啟動排程器並註冊所有任務
func StartScheduler(service *app.AppServices) {
	log := logger.Logger
	cronLogger := &logger.LogrusAdapter{Logger: log}

	//Scheduler = cron.New(cron.WithSeconds()) // 支援秒級調度
	Scheduler = cron.New(cron.WithChain(cron.SkipIfStillRunning(cronLogger)))

	// 建立每分鐘檢查任務狀態任務
	_, err := Scheduler.AddFunc("@every 1m", func() {
		tasks, err := service.TaskServices.ListTasks(map[string]interface{}{
			"status": models.TASK_STATUS_ACTIVE,
		}, "created_at", "ASC")
		if err != nil {
			errMsg := fmt.Sprintf("TaskServices.ListTasks err: %v", err)
			log.Error(errMsg)
			return
		}
		for _, task := range tasks {
			switch task.Type {
			case "http":

			}
		}
	})
	if err != nil {
		logger.Logger.WithError(err).Error("Failed to scheduler cleanup task")
	}

	Scheduler.Start()
}

// StopScheduler 停止排程器
func StopScheduler() {
	if Scheduler != nil {
		Scheduler.Stop()
		logger.Logger.Info("排程器已停止")
	}
}
