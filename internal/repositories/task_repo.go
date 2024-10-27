package repositories

import (
	"github.com/q90016200/schedule-go/internal/models"
	"gorm.io/gorm"
)

// TaskRepository 任何型別，只要符合定義規則的話，就可以被納入 TaskRepository interface
type TaskRepository interface {
	// Create 這個函式需要接收兩個參數（string, int），並回傳 (string, error) 才符合入會資格
	Create(task *models.Task) error
}

type taskRepository struct {
	db *gorm.DB
}

// NewTaskRepository 建立 TaskRepository 實例
func NewTaskRepository(db *gorm.DB) TaskRepository {
	return &taskRepository{db: db}
}

// Create 建立一個新的 Task
func (r *taskRepository) Create(task *models.Task) error {
	return r.db.Create(task).Error
}
