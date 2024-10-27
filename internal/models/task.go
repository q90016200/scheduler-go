package models

// Task 任務表 (tasks)
type Task struct {
	ID          string `json:"id" gorm:"primaryKey;autoIncrement"`
	GroupID     string `json:"group_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Schedule    string `json:"schedule"`
	Type        string `json:"type"`
	Endpoint    string `json:"endpoint"`
	Method      string `json:"method"`
	Payload     string `json:"payload"`
	Status      string `json:"status"`
	CreatedBy   string `json:"created_by"`
	UpdatedBy   string `json:"updated_by"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}

// TaskGroup 群組表 (groups)
type TaskGroup struct {
	ID          string `json:"id" gorm:"primaryKey;autoIncrement"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"` // 群組狀態 ('active', 'inactive')
	CreatedBy   string `json:"created_by"`
	UpdatedBy   string `json:"updated_by"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}
