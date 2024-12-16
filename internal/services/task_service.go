package services

import (
	"fmt"
	"github.com/q90016200/scheduler-go/internal/models"
	"github.com/q90016200/scheduler-go/internal/repositories"
	"github.com/q90016200/scheduler-go/pkg/logger"
	"github.com/rs/xid"
	"sync"
)

type TaskService struct {
	enableTasks *sync.Map
	repo        *repositories.RepositoryManager
}

func NewTaskService(repo *repositories.RepositoryManager) *TaskService {
	return &TaskService{
		repo: repo,
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
	//if task.Status == models.TASK_STATUS_ACTIVE {
	//	err := s.CreateCronTask(task.ID, task)
	//	if err != nil {
	//		return nil, err
	//	}
	//}

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
		logger.Logger.Error(errMsg)
		return nil, err
	}

	// 重新啟動排程任務
	//if err := s.DeleteCronTask(id); err != nil {
	//	errMsg := fmt.Sprintf("task: %T, DeleteCronTask err: %v", task, err)
	//	logger.Logger.Error(errMsg)
	//	return nil, err
	//}
	//if err := s.CreateCronTask(id, task); err != nil {
	//	errMsg := fmt.Sprintf("task: %T, CreateCronTask err: %v", task, err)
	//	logger.Logger.Error(errMsg)
	//	return nil, err
	//}

	return task, nil
}

func (s *TaskService) DeleteTask(id string) error {
	if err := s.repo.TaskRepo.Delete(id); err != nil {
		errMsg := fmt.Sprintf("task id: %s, repo.TaskRepo.Delete err: %v", id, err)
		logger.Logger.Error(errMsg)
		return err
	}

	//if err := s.DeleteCronTask(id); err != nil {
	//	errMsg := fmt.Sprintf("task id: %s, DeleteCronTask err: %v", id, err)
	//	logger.Logger.Error(errMsg)
	//	return err
	//}

	return nil
}
