package services

import (
	"errors"
	"fmt"
	"github.com/q90016200/scheduler-go/internal/models"
	"github.com/q90016200/scheduler-go/internal/repositories"
	"github.com/q90016200/scheduler-go/internal/scheduler"
	"github.com/q90016200/scheduler-go/pkg/logger"
	"github.com/rs/xid"
)

type TaskService struct {
	repo          *repositories.RepositoryManager
	cronScheduler *scheduler.CronScheduler
}

func NewTaskService(repo *repositories.RepositoryManager, cronScheduler *scheduler.CronScheduler) *TaskService {
	return &TaskService{
		cronScheduler: cronScheduler,
		repo:          repo,
	}
}

// ListTasks 支持篩選與排序
func (s *TaskService) ListTasks(filters map[string]interface{}, sortField, sortOrder string) ([]models.Task, error) {
	return s.repo.TaskRepo.ListTasks(filters, sortField, sortOrder)
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

	if err := s.repo.TaskRepo.Create(task); err != nil {
		return nil, err
	}

	// 當是執行的放入 syncMap 內，供排程取得目前可執行 task 清單
	if task.Status == models.TASK_STATUS_ACTIVE {
		err := scheduler.AddJob(s.cronScheduler, task.ID, task)
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

	if err := s.repo.TaskRepo.Update(task); err != nil {
		errMsg := fmt.Sprintf("task: %T, repo.TaskRepo.Update err: %v", task, err)
		return nil, errors.New(errMsg)
	}

	// 重新啟動排程任務
	if err := scheduler.RemoveJob(s.cronScheduler, id); err != nil {
		errMsg := fmt.Sprintf("task: %T, RemoveJob err: %v", task, err)
		logger.Log.Error(errMsg)
		return nil, err
	}
	if err := scheduler.AddJob(s.cronScheduler, id, task); err != nil {
		errMsg := fmt.Sprintf("task: %T, AddJob err: %v", task, err)
		logger.Log.Error(errMsg)
		return nil, err
	}

	return task, nil
}

func (s *TaskService) DeleteTask(id string) error {
	if err := s.repo.TaskRepo.Delete(id); err != nil {
		errMsg := fmt.Sprintf("task id: %s, repo.TaskRepo.Delete err: %v", id, err)
		logger.Log.Error(errMsg)
		return err
	}

	if err := scheduler.RemoveJob(s.cronScheduler, id); err != nil {
		errMsg := fmt.Sprintf("task id: %s, RemoveJob err: %v", id, err)
		logger.Log.Error(errMsg)
		return err
	}

	return nil
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

	if err := s.repo.TaskRepo.CreateGroup(group); err != nil {
		return nil, err
	}

	return group, nil
}
