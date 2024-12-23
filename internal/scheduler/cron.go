package scheduler

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/q90016200/scheduler-go/internal/models"
	"github.com/q90016200/scheduler-go/internal/repositories"
	"github.com/q90016200/scheduler-go/pkg/logger"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
	"sync"
)

// CronScheduler 全局的排程器實例
type CronScheduler struct {
	Cron        *cron.Cron
	Log         *logger.LogrusAdapter
	jobIDs      map[string]cron.EntryID // 任務名稱與 EntryID 的映射
	mu          sync.Mutex              // 保證多協程安全
	repo        *repositories.RepositoryManager
	restyClient *resty.Client
}

func NewCronScheduler(db *gorm.DB) *CronScheduler {
	log := logger.Logger
	cronLogger := &logger.LogrusAdapter{Logger: log}
	repoManager := repositories.NewRepositoryManager(db)
	return &CronScheduler{
		Cron:        cron.New(cron.WithChain(cron.SkipIfStillRunning(cronLogger))),
		Log:         cronLogger,
		jobIDs:      make(map[string]cron.EntryID),
		repo:        repoManager,
		restyClient: resty.New(),
	}
}

// StartScheduler 啟動排程器並註冊所有任務
func (cs *CronScheduler) StartScheduler() {
	// 建立每分鐘檢查任務狀態任務
	_, err := cs.Cron.AddFunc("@every 1m", func() {
		tasks, err := cs.repo.TaskRepo.ListTasks(map[string]interface{}{
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
					err = AddJob(cs, task.ID, &task)
					if err != nil {
						logger.Logger.Info("task ID: " + task.ID + ", AddJob err:" + err.Error())
					}
				}
			} else {
				if exists {
					err = RemoveJob(cs, task.ID)
					if err != nil {
						logger.Logger.Info("task ID: " + task.ID + ", RemoveJob err:" + err.Error())
					}
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
