package services

import (
	"errors"
	"fmt"
	"github.com/q90016200/scheduler-go/internal/models"
	"github.com/q90016200/scheduler-go/internal/scheduler"
	"github.com/q90016200/scheduler-go/pkg/logger"
	"github.com/rs/xid"
	"gorm.io/gorm"
)

type TaskService struct {
	db            *gorm.DB
	cronScheduler *scheduler.CronScheduler
}

func NewTaskService(db *gorm.DB, cronScheduler *scheduler.CronScheduler) *TaskService {
	return &TaskService{
		cronScheduler: cronScheduler,
		db:            db,
	}
}

// listTasksSortableFields 排序欄位白名單，避免 sortField 直接拼進 SQL 造成 injection
var listTasksSortableFields = map[string]bool{
	"name":       true,
	"created_at": true,
	"status":     true,
	"schedule":   true,
}

// ListTasks 支持篩選、排序與分頁
func (s *TaskService) ListTasks(status, groupID, name, sortField, sortOrder string, limit, offset int) ([]models.Task, int64, error) {
	var tasks []models.Task
	query := s.db.Model(&models.Task{})

	// 篩選條件
	if status != "" {
		query = query.Where("tasks.status = ?", status)
	}
	if groupID != "" {
		query = query.Where("tasks.group_id = ?", groupID)
	}
	if name != "" {
		query = query.Where("tasks.name ILIKE ?", "%"+name+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 排序條件
	if listTasksSortableFields[sortField] {
		order := "ASC"
		if sortOrder == "desc" {
			order = "DESC"
		}
		query = query.Order(sortField + " " + order)
	}

	// 查詢結果
	err := query.Preload("Group").Limit(limit).Offset(offset).Find(&tasks).Error
	return tasks, total, err
}

// GetTask 取得單一任務
func (s *TaskService) GetTask(id string) (*models.Task, error) {
	var task models.Task
	err := s.db.Preload("Group").First(&task, "id = ?", id).Error
	return &task, err
}

func (s *TaskService) CreateTask(input models.CreateTaskReq) (*models.Task, error) {
	guid := xid.New()
	task := &models.Task{
		ID:          guid.String(),
		GroupID:     input.GroupID,
		Name:        input.Name,
		Description: input.Description,
		Schedule:    input.Schedule,
		Type:        input.Type,
		Endpoint:    input.EndPoint,
		Method:      input.Method,
		Payload:     input.Payload,
		Status:      input.Status,
		CreatedBy:   input.UserId,
		UpdatedBy:   input.UserId,
	}

	if err := s.db.Create(task).Error; err != nil {
		return nil, err
	}

	// 當是執行的放入 syncMap 內，供排程取得目前可執行 task 清單
	if task.Status == models.TASK_STATUS_ACTIVE {
		err := scheduler.RegisterTaskJob(s.cronScheduler, task.ID, task)
		if err != nil {
			return nil, err
		}
	}

	return task, nil
}

func (s *TaskService) UpdateTask(id string, input models.UpdateTaskReq) (*models.Task, error) {
	task := &models.Task{
		ID:          id,
		GroupID:     input.GroupID,
		Name:        input.Name,
		Description: input.Description,
		Schedule:    input.Schedule,
		Type:        input.Type,
		Endpoint:    input.EndPoint,
		Method:      input.Method,
		Payload:     input.Payload,
		Status:      input.Status,
		UpdatedBy:   input.UserId,
	}

	if err := s.db.Omit("CreatedAt", "CreatedBy").Save(task).Error; err != nil {
		errMsg := fmt.Sprintf("task: %T, db.Save err: %v", task, err)
		return nil, errors.New(errMsg)
	}

	// 重新啟動排程任務
	if err := scheduler.UnregisterTaskJob(s.cronScheduler, id); err != nil {
		errMsg := fmt.Sprintf("task: %T, UnregisterTaskJob err: %v", task, err)
		logger.Log.Error(errMsg)
		return nil, err
	}
	if task.Status == models.TASK_STATUS_ACTIVE {
		if err := scheduler.RegisterTaskJob(s.cronScheduler, id, task); err != nil {
			errMsg := fmt.Sprintf("task: %T, RegisterTaskJob err: %v", task, err)
			logger.Log.Error(errMsg)
			return nil, err
		}
	}

	return task, nil
}

// RunTaskNow 立即執行一次任務，不受排程狀態影響
func (s *TaskService) RunTaskNow(id string) (*models.Task, error) {
	task, err := s.GetTask(id)
	if err != nil {
		return nil, err
	}
	if err := scheduler.RunTaskNow(s.cronScheduler, task); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *TaskService) DeleteTask(id string) error {
	if err := s.db.Where("id = ?", id).Delete(&models.Task{}).Error; err != nil {
		errMsg := fmt.Sprintf("task id: %s, db.Delete err: %v", id, err)
		logger.Log.Error(errMsg)
		return err
	}

	if err := scheduler.UnregisterTaskJob(s.cronScheduler, id); err != nil {
		errMsg := fmt.Sprintf("task id: %s, UnregisterTaskJob err: %v", id, err)
		logger.Log.Error(errMsg)
		return err
	}

	return nil
}

// ListTaskGroups 列出所有任務群組，支援分頁
func (s *TaskService) ListTaskGroups(limit, offset int) ([]models.TaskGroup, int64, error) {
	var groups []models.TaskGroup
	query := s.db.Model(&models.TaskGroup{})

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Limit(limit).Offset(offset).Find(&groups).Error
	return groups, total, err
}

// GetTaskGroup 取得單一任務群組
func (s *TaskService) GetTaskGroup(id string) (*models.TaskGroup, error) {
	var group models.TaskGroup
	err := s.db.First(&group, "id = ?", id).Error
	return &group, err
}

func (s *TaskService) CreateTaskGroup(input models.CreateTaskGroupReq) (*models.TaskGroup, error) {
	group := &models.TaskGroup{
		ID:          xid.New().String(),
		Name:        input.Name,
		Description: input.Description,
		Status:      input.Status,
		CreatedBy:   input.UserId,
		UpdatedBy:   input.UserId,
	}

	if err := s.db.Create(group).Error; err != nil {
		return nil, err
	}

	return group, nil
}

func (s *TaskService) UpdateTaskGroup(id string, input models.UpdateTaskGroupReq) (*models.TaskGroup, error) {
	group := &models.TaskGroup{
		ID:          id,
		Name:        input.Name,
		Description: input.Description,
		Status:      input.Status,
		UpdatedBy:   input.UserId,
	}

	if err := s.db.Omit("CreatedAt", "CreatedBy").Save(group).Error; err != nil {
		errMsg := fmt.Sprintf("task: %T, db.Save err: %v", group, err)
		return nil, errors.New(errMsg)
	}

	// 群組停用時連動停用底下仍為 active 的任務並取消排程；改回啟用不自動恢復子任務
	if input.Status == models.TASK_STATUS_INACTIVE {
		var tasks []models.Task
		if err := s.db.Where("group_id = ? AND status = ?", id, models.TASK_STATUS_ACTIVE).Find(&tasks).Error; err != nil {
			return nil, err
		}
		if len(tasks) > 0 {
			if err := s.db.Model(&models.Task{}).
				Where("group_id = ? AND status = ?", id, models.TASK_STATUS_ACTIVE).
				Update("status", models.TASK_STATUS_INACTIVE).Error; err != nil {
				return nil, err
			}
			for _, task := range tasks {
				if err := scheduler.UnregisterTaskJob(s.cronScheduler, task.ID); err != nil {
					logger.Log.Error(fmt.Sprintf("group id: %s, task id: %s, UnregisterTaskJob err: %v", id, task.ID, err))
				}
			}
		}
	}

	return group, nil
}

func (s *TaskService) DeleteTaskGroup(id string) error {
	if err := s.db.Where("id = ?", id).Delete(&models.TaskGroup{}).Error; err != nil {
		errMsg := fmt.Sprintf("task id: %s, db.Delete err: %v", id, err)
		logger.Log.Error(errMsg)
		return err
	}

	return nil
}
