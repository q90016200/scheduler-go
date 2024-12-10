package repositories

import (
	"github.com/q90016200/schedule-go/internal/models"
	"gorm.io/gorm"
)

// TaskRepository 任何型別，只要符合定義規則的話，就可以被納入 TaskRepository interface
type TaskRepository interface {
	// Create 這個函式需要接收兩個參數（string, int），並回傳 (string, error) 才符合入會資格
	Create(task *models.Task) error
	Update(task *models.Task) error
	Delete(id string) error
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
	// 預設寫入時間，已在 gorm tag 自動寫入
	//task.CreatedAt = time.Now().Unix()
	//task.UpdatedAt = time.Now().Unix()
	return r.db.Create(task).Error
}

func (r *taskRepository) Update(task *models.Task) error {
	return r.db.Save(task).Error
}

func (r *taskRepository) Delete(id string) error {
	return r.db.Where("id = ?", id).Delete(&models.Task{}).Error
}
