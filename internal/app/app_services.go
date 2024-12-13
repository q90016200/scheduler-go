package app

import (
	"github.com/q90016200/schedule-go/internal/repositories"
	"github.com/q90016200/schedule-go/internal/services"
	"gorm.io/gorm"
	"sync"
)

type AppServices struct {
	EnabledTasks *sync.Map // 共享變數存放所有可啟用的任務
	// Service
	TaskServices *services.TaskService
	AuthService  *services.AuthService
}

func NewAppServices(db *gorm.DB) *AppServices {
	repoManager := repositories.NewRepositoryManager(db)
	enableTasks := new(sync.Map)
	return &AppServices{
		EnabledTasks: enableTasks, // 初始化共享變數
		TaskServices: services.NewTaskService(enableTasks, repoManager),
		AuthService:  services.NewAuthService(repoManager),
	}
}
