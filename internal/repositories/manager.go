package repositories

import "gorm.io/gorm"

type RepositoryManager struct {
	TaskRepo TaskRepository
	UserRepo UserRepository
	// 其他 Repository 可以繼續添加
}

// NewRepositoryManager 初始化並返回 RepositoryManager
func NewRepositoryManager(db *gorm.DB) *RepositoryManager {
	return &RepositoryManager{
		TaskRepo: NewTaskRepository(db),
		UserRepo: NewUserRepository(db),
		// 其他 Repository 初始化
	}
}
