package services

import (
	"github.com/q90016200/scheduler-go/internal/models"
	"gorm.io/gorm"
)

type TaskExecutionLogService struct {
	db *gorm.DB
}

func NewTaskExecutionLogService(db *gorm.DB) *TaskExecutionLogService {
	return &TaskExecutionLogService{db: db}
}

// ListLogs 依 task_id 篩選（可留空查全部），依建立時間新到舊排序，limit/offset 分頁
func (s *TaskExecutionLogService) ListLogs(taskID string, limit, offset int) ([]models.TaskExecutionLog, int64, error) {
	var logs []models.TaskExecutionLog
	query := s.db.Model(&models.TaskExecutionLog{})
	if taskID != "" {
		query = query.Where("task_id = ?", taskID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&logs).Error
	return logs, total, err
}

func (s *TaskExecutionLogService) GetLog(id string) (*models.TaskExecutionLog, error) {
	var l models.TaskExecutionLog
	err := s.db.First(&l, "id = ?", id).Error
	return &l, err
}
