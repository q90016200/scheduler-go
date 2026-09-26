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

	RetryCount          int    `json:"retry_count" gorm:"column:retry_count;default:0"`                             // 失敗時額外重試次數 (0-5)
	RetryDelayMs        int    `json:"retry_delay_ms" gorm:"column:retry_delay_ms;default:0"`                       // 每次重試間隔，固定值不做指數退避 (0-30000ms)
	TimeoutSeconds      int    `json:"timeout_seconds" gorm:"column:timeout_seconds;default:10"`                    // http/rpc 單次呼叫逾時秒數 (1-120)
	Headers             string `json:"headers" gorm:"column:headers;default:'{}'"`                                  // 自由 JSON map，如 {"Authorization":"Bearer xxx"}；跟 Payload/Endpoint 同信任模型，明文存
	SuccessStatusCodes  string `json:"success_status_codes" gorm:"column:success_status_codes;size:100;default:''"` // 只用於 http 任務；如 "200,201,2xx"，空字串沿用預設（<400 成功）
	NotifyFailThreshold int    `json:"notify_fail_threshold" gorm:"column:notify_fail_threshold;default:0"`         // 連續失敗達此值才通知一次，之後靜音直到成功一次重新計數；0 表示不啟用
	NotifyTelegram      bool   `json:"notify_telegram" gorm:"column:notify_telegram;default:false"`
	NotifySlack         bool   `json:"notify_slack" gorm:"column:notify_slack;default:false"`
	ConsecutiveFailures int    `json:"consecutive_failures" gorm:"column:consecutive_failures;default:0"` // 只計 cron 來源的最終失敗；成功或發過通知後清零；跨重啟/停用重啟都保留

	NextRunAt *int64 `json:"next_run_at,omitempty" gorm:"-"` // unix 秒，由 service 層從 cron entry 裝飾，只有已排程且 active 的任務才有值
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
	GroupID             *string `json:"group_id"`
	Name                string  `json:"name" validate:"required"`
	Description         string  `json:"description"`
	Schedule            string  `json:"schedule" validate:"required,cron"`
	Type                string  `json:"type" validate:"required,oneof=http rpc"`
	EndPoint            string  `json:"endpoint" validate:"required"`
	Method              string  `json:"method"`
	Payload             string  `json:"payload"`
	Status              string  `json:"status" validate:"omitempty,oneof=enabled disabled"`
	UserId              string  `json:"user_id"`
	RetryCount          int     `json:"retry_count" validate:"omitempty,min=0,max=5"`
	RetryDelayMs        int     `json:"retry_delay_ms" validate:"omitempty,min=0,max=30000"`
	TimeoutSeconds      int     `json:"timeout_seconds" validate:"omitempty,min=1,max=120"`
	Headers             string  `json:"headers" validate:"omitempty,json"`
	SuccessStatusCodes  string  `json:"success_status_codes" validate:"omitempty,max=100,status_codes"`
	NotifyFailThreshold int     `json:"notify_fail_threshold" validate:"omitempty,min=1,max=20"`
	NotifyTelegram      bool    `json:"notify_telegram"`
	NotifySlack         bool    `json:"notify_slack"`
}

// UpdateTaskReq 欄位與 CreateTaskReq 完全相同，直接設為型別別名
type UpdateTaskReq = CreateTaskReq

type CreateTaskGroupReq struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
	Status      string `json:"status" validate:"omitempty,oneof=enabled disabled"`
	UserId      string `json:"user_id"`
}

// UpdateTaskGroupReq 欄位與 CreateTaskGroupReq 完全相同，直接設為型別別名
type UpdateTaskGroupReq = CreateTaskGroupReq

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
