package models

// STATUS_ENABLED/STATUS_DISABLED 共用於 Task、TaskGroup、User 的狀態欄位
const STATUS_ENABLED = "enabled"
const STATUS_DISABLED = "disabled"

// Task 任務表 (tasks)
type Task struct {
	ID          string     `json:"id" gorm:"column:id;primaryKey;size:20"`
	GroupID     *string    `json:"group_id" gorm:"column:group_id;size:20;default:null"` // 外鍵
	Name        string     `json:"name" gorm:"column:name;size:100"`
	Description string     `json:"description" gorm:"column:description"`
	Schedule    string     `json:"schedule" gorm:"column:schedule"`
	Type        string     `json:"type" gorm:"column:type;size:20"`
	Endpoint    string     `json:"endpoint" gorm:"column:endpoint"`
	Method      string     `json:"method" gorm:"column:method;size:255"`
	Payload     string     `json:"payload" gorm:"column:payload"`
	Status      string     `json:"status" gorm:"column:status;size:10"` // 狀態 ('enabled', 'disabled')
	CreatedBy   string     `json:"created_by" gorm:"column:created_by;size:20"`
	UpdatedBy   string     `json:"updated_by" gorm:"column:updated_by;size:20"`
	CreatedAt   int64      `json:"created_at" gorm:"column:created_at;autoCreateTime;<-:create"`
	UpdatedAt   int64      `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	Group       *TaskGroup `json:"group" gorm:"foreignKey:GroupID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"` // 關聯設置
}

// TaskGroup 群組表 (task_groups)
type TaskGroup struct {
	ID          string `json:"id" gorm:"primaryKey;size:20"`
	Name        string `json:"name" gorm:"size:50"`
	Description string `json:"description"`
	Status      string `json:"status" gorm:"not null;size:10"` // 群組狀態 ('enabled', 'disabled')
	CreatedBy   string `json:"created_by" gorm:"size:20"`
	UpdatedBy   string `json:"updated_by" gorm:"size:20"`
	CreatedAt   int64  `json:"created_at" gorm:"autoCreateTime;<-:create"`
	UpdatedAt   int64  `json:"updated_at" gorm:"autoUpdateTime"`
}

type CreateTaskReq struct {
	GroupID     *string `json:"group_id"`
	Name        string  `json:"name" validate:"required"`
	Description string  `json:"description"`
	Schedule    string  `json:"schedule" validate:"required,cron"`
	Type        string  `json:"type" validate:"required,oneof=http rpc"`
	EndPoint    string  `json:"endpoint" validate:"required"`
	Method      string  `json:"method"`
	Payload     string  `json:"payload"`
	Status      string  `json:"status" validate:"omitempty,oneof=enabled disabled"`
	UserId      string  `json:"user_id"`
}

type UpdateTaskReq struct {
	GroupID     *string `json:"group_id"`
	Name        string  `json:"name" validate:"required"`
	Description string  `json:"description"`
	Schedule    string  `json:"schedule" validate:"required,cron"`
	Type        string  `json:"type" validate:"required,oneof=http rpc"`
	EndPoint    string  `json:"endpoint" validate:"required"`
	Method      string  `json:"method"`
	Payload     string  `json:"payload"`
	Status      string  `json:"status" validate:"omitempty,oneof=enabled disabled"`
	UserId      string  `json:"user_id"`
}

type CreateTaskGroupReq struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
	Status      string `json:"status" validate:"omitempty,oneof=enabled disabled"`
	UserId      string `json:"user_id"`
}

type UpdateTaskGroupReq struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
	Status      string `json:"status" validate:"omitempty,oneof=enabled disabled"`
	UserId      string `json:"user_id"`
}

// UpdateTaskStatusReq 只切換任務啟用/停用狀態，不需送整包任務資料
type UpdateTaskStatusReq struct {
	Status string `json:"status" validate:"required,oneof=enabled disabled"`
	UserId string `json:"user_id"`
}

// RunTaskRequest 立即執行任務的參數；Repeat/IntervalMs 為 0（未帶 body 或欄位省略）時
// 由 controller 補上預設值 1 次 / 1000ms，不當成驗證錯誤
type RunTaskRequest struct {
	Repeat     int `json:"repeat" validate:"omitempty,min=1,max=50"`
	IntervalMs int `json:"interval_ms" validate:"omitempty,min=100,max=60000"`
}
