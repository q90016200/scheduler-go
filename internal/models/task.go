package models

// Task 任務表 (tasks)
type Task struct {
	ID          uint              `json:"id" gorm:"primaryKey;autoIncrement"`
	GroupID     uint              `json:"group_id"`
	Name        string            `json:"name" gorm:"size=100" validate:"required"`
	Description string            `json:"description"`
	Schedule    string            `json:"schedule"`
	Type        string            `json:"type" gorm:"size:20"`
	Endpoint    string            `json:"endpoint"`
	Method      string            `json:"method" gorm:"size:10"`
	Payload     map[string]string `json:"payload"`
	Status      string            `json:"status" gorm:"size:10"` // 群組狀態 ('active', 'inactive')
	CreatedBy   string            `json:"created_by"`
	UpdatedBy   string            `json:"updated_by"`
	CreatedAt   int64             `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   int64             `json:"updated_at" gorm:"autoUpdateTime"`
	Group       TaskGroup         `gorm:"foreignKey:GroupID"`
}

// TaskGroup 群組表 (groups)
type TaskGroup struct {
	ID          uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	Name        string `json:"name" gorm:"size:50"`
	Description string `json:"description"`
	Status      string `json:"status" gorm:"not null;size=10"` // 群組狀態 ('active', 'inactive')
	CreatedBy   string `json:"created_by"`
	UpdatedBy   string `json:"updated_by"`
	CreatedAt   int64  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   int64  `json:"updated_at" gorm:"autoUpdateTime"`
}
