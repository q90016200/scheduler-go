package repositories

import "gorm.io/gorm"

type RepositoryManager struct {
	TaskRepository TaskRepository
	// 其他 Repository 可以繼續添加
}

// NewRepositoryManager 初始化並返回 RepositoryManager
func NewRepositoryManager(db *gorm.DB) *RepositoryManager {
	return &RepositoryManager{
		TaskRepository: NewTaskRepository(db),
		// 其他 Repository 初始化
	}
}
