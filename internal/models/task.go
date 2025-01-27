package models

const TASK_STATUS_ACTIVE = "active"
const TASK_STATUS_INACTIVE = "inactive"

// Task 任務表 (tasks)
type Task struct {
	ID          string    `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	GroupID     *string   `json:"group_id" gorm:"column:group_id;default:null"`
	Name        string    `json:"name" gorm:"column:name;size=100"`
	Description string    `json:"description" gorm:"column:description"`
	Schedule    string    `json:"schedule" gorm:"column:schedule"`
	Type        string    `json:"type" gorm:"column:type;size:20"`
	Endpoint    string    `json:"endpoint" gorm:"column:endpoint"`
	Method      string    `json:"method" gorm:"column:method;size:10"`
	Payload     string    `json:"payload" gorm:"column:payload"`
	Status      string    `json:"status" gorm:"column:status;size:10"` // 群組狀態 ('active', 'inactive')
	CreatedBy   string    `json:"created_by" gorm:"column:created_by"`
	UpdatedBy   string    `json:"updated_by" gorm:"column:updated_by"`
	CreatedAt   int64     `json:"created_at" gorm:"column:created_at;autoCreateTime;<-:create"`
	UpdatedAt   int64     `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	Group       TaskGroup `gorm:"foreignKey:GroupID"`
}

// TaskGroup 群組表 (groups)
type TaskGroup struct {
	ID          string `json:"id" gorm:"primaryKey;autoIncrement"`
	Name        string `json:"name" gorm:"size:50"`
	Description string `json:"description"`
	Status      string `json:"status" gorm:"not null;size=10"` // 群組狀態 ('active', 'inactive')
	CreatedBy   string `json:"created_by"`
	UpdatedBy   string `json:"updated_by"`
	CreatedAt   int64  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   int64  `json:"updated_at" gorm:"autoUpdateTime"`
}

type CreateTaskReq struct {
	GroupID     *string `json:"group_id"`
	Name        string  `json:"name" validate:"required"`
	Description string  `json:"description"`
	Schedule    string  `json:"schedule" validate:"required"`
	Type        string  `json:"type" validate:"required" `
	EndPoint    string  `json:"endpoint" validate:"required"`
	Method      string  `json:"method" validate:"omitempty,required_if=Type HTTP,oneof=GET POST PUT DELETE PATCH"`
	Payload     string  `json:"payload"`
	Status      string  `json:"status" validate:"omitempty,oneof=active inactive"`
	UserId      string  `json:"user_id"`
}

type UpdateTaskReq struct {
	GroupID     *string `json:"group_id"`
	Name        string  `json:"name" validate:"required"`
	Description string  `json:"description"`
	Schedule    string  `json:"schedule" validate:"required"`
	Type        string  `json:"type" validate:"required" `
	EndPoint    string  `json:"endpoint" validate:"required"`
	Method      string  `json:"method" validate:"omitempty,required_if=Type HTTP,oneof=GET POST PUT DELETE PATCH"`
	Payload     string  `json:"payload"`
	Status      string  `json:"status" validate:"omitempty,oneof=active inactive"`
	UserId      string  `json:"user_id"`
}
