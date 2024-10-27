package services

import (
	"github.com/q90016200/schedule-go/internal/repositories"
)

type TaskService struct {
	repo *repositories.RepositoryManager
}

func NewTaskService(repo *repositories.RepositoryManager) *TaskService {
	return &TaskService{repo: repo}
}

func (s TaskService) CreateTask() {
	//s.repo.TaskRepository.Create()
}
