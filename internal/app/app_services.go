package app

import (
	"github.com/q90016200/scheduler-go/internal/scheduler"
	"github.com/q90016200/scheduler-go/internal/services"
	"gorm.io/gorm"
)

type AppServices struct {
	//EnabledTasks *sync.Map // 共享變數存放所有可啟用的任務
	// Service
	TaskServices            *services.TaskService
	AuthService             *services.AuthService
	TaskExecutionLogService *services.TaskExecutionLogService
}

func NewAppServices(db *gorm.DB, cronScheduler *scheduler.CronScheduler) *AppServices {
	//enableTasks := new(sync.Map)
	return &AppServices{
		//EnabledTasks: enableTasks, // 初始化共享變數
		TaskServices:            services.NewTaskService(db, cronScheduler),
		AuthService:             services.NewAuthService(db),
		TaskExecutionLogService: services.NewTaskExecutionLogService(db),
	}
}
