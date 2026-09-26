package app

import (
	"github.com/q90016200/scheduler-go/internal/scheduler"
	"github.com/q90016200/scheduler-go/internal/services"
	"gorm.io/gorm"
)

type AppServices struct {
	TaskServices            *services.TaskService
	AuthService             *services.AuthService
	TaskExecutionLogService *services.TaskExecutionLogService
	UserService             *services.UserService
}

func NewAppServices(db *gorm.DB, cronScheduler *scheduler.CronScheduler) *AppServices {
	return &AppServices{
		TaskServices:            services.NewTaskService(db, cronScheduler),
		AuthService:             services.NewAuthService(db),
		TaskExecutionLogService: services.NewTaskExecutionLogService(db),
		UserService:             services.NewUserService(db),
	}
}
