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

// ListTasks 支持篩選與排序
func (s *TaskService) ListTasks(status, sortField, sortOrder string) ([]models.Task, error) {
	var tasks []models.Task
	query := s.db

	// 篩選條件
	if status != "" {
		query = query.Where("tasks.status = ?", status)
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
	err := query.Preload("Group").Find(&tasks).Error
	return tasks, err
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
		CreatedBy:   input.UserId,
		UpdatedBy:   input.UserId,
	}

	if err := s.db.Omit("CreatedAt").Save(task).Error; err != nil {
		errMsg := fmt.Sprintf("task: %T, db.Save err: %v", task, err)
		return nil, errors.New(errMsg)
	}

	// 重新啟動排程任務
	if err := scheduler.UnregisterTaskJob(s.cronScheduler, id); err != nil {
		errMsg := fmt.Sprintf("task: %T, UnregisterTaskJob err: %v", task, err)
		logger.Log.Error(errMsg)
		return nil, err
	}
	if err := scheduler.RegisterTaskJob(s.cronScheduler, id, task); err != nil {
		errMsg := fmt.Sprintf("task: %T, RegisterTaskJob err: %v", task, err)
		logger.Log.Error(errMsg)
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

// ListTaskGroups 列出所有任務群組
func (s *TaskService) ListTaskGroups() ([]models.TaskGroup, error) {
	var groups []models.TaskGroup
	err := s.db.Find(&groups).Error
	return groups, err
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

	if err := s.db.Omit("CreatedAt").Save(group).Error; err != nil {
		errMsg := fmt.Sprintf("task: %T, db.Save err: %v", group, err)
		return nil, errors.New(errMsg)
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
