package services

import (
	"github.com/q90016200/schedule-go/internal/models"
	"github.com/q90016200/schedule-go/internal/repositories"
	"sync"
)

type TaskService struct {
	enableTasks *sync.Map
	repo        *repositories.RepositoryManager
}

func NewTaskService(enableTasks *sync.Map, repo *repositories.RepositoryManager) *TaskService {
	return &TaskService{
		enableTasks: enableTasks,
		repo:        repo,
	}
}

func (s TaskService) CreateTask(input models.CreateTaskReq) (*models.Task, error) {
	task := &models.Task{
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
		err := s.CreateCronTask(task.ID, task)
		if err != nil {
			return nil, err
		}
	}

	return task, nil
}

func (s *TaskService) UpdateTask() (*models.Task, error) {
	return nil, nil
}

func (s *TaskService) DeleteTask(id uint) error {
	return nil
}

func (s *TaskService) CreateCronTask(id uint, task *models.Task) error {
	//s.enableTasks.Store(task.ID, task)
	return nil
}

func (s TaskService) DeleteCronTask(id uint) error {
	return nil
}
