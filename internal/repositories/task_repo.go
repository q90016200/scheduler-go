package repositories

import (
	"github.com/q90016200/scheduler-go/internal/models"
	"gorm.io/gorm"
)

// TaskRepository 任何型別，只要符合定義規則的話，就可以被納入 TaskRepository interface
type TaskRepository interface {
	ListTasks(filters map[string]interface{}, sortField string, sortOrder string) ([]models.Task, error)
	Create(task *models.Task) error
	Update(task *models.Task) error
	Delete(id string) error
	CreateGroup(tg *models.TaskGroup) error
	UpdateGroup(task *models.TaskGroup) error
	DeleteGroup(id string) error
}

type taskRepository struct {
	db *gorm.DB
}

// NewTaskRepository 建立 TaskRepository 實例
func NewTaskRepository(db *gorm.DB) TaskRepository {
	return &taskRepository{db: db}
}

func (r *taskRepository) ListTasks(filters map[string]interface{}, sortField string, sortOrder string) ([]models.Task, error) {
	var tasks []models.Task
	query := r.db

	// 篩選條件
	if status, ok := filters["status"]; ok {
		query = query.Where("tasks.status = ?", status)
	}

	// 排序條件
	if sortField != "" {
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

// Create 建立一個新的 Task
func (r *taskRepository) Create(task *models.Task) error {
	// 預設寫入時間，已在 gorm tag 自動寫入
	//task.CreatedAt = time.Now().Unix()
	//task.UpdatedAt = time.Now().Unix()
	return r.db.Create(task).Error
}

func (r *taskRepository) Update(task *models.Task) error {
	return r.db.Save(task).Omit("CreatedAt").Error
}

func (r *taskRepository) Delete(id string) error {
	return r.db.Where("id = ?", id).Delete(&models.Task{}).Error
}

func (r *taskRepository) CreateGroup(tg *models.TaskGroup) error {
	return r.db.Create(tg).Error
}

func (r *taskRepository) UpdateGroup(tg *models.TaskGroup) error {
	return r.db.Save(tg).Omit("CreatedAt").Error
}

func (r *taskRepository) DeleteGroup(id string) error {
	return r.db.Where("id = ?", id).Delete(&models.TaskGroup{}).Error
}
