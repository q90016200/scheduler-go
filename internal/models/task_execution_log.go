package models

const EXEC_STATUS_SUCCESS = "success"
const EXEC_STATUS_FAILED = "failed"

// TaskExecutionLog 任務執行紀錄 (task_execution_logs)
type TaskExecutionLog struct {
	ID         string `json:"id" gorm:"column:id;primaryKey;size:20"`
	TaskID     string `json:"task_id" gorm:"column:task_id;size:20;index"`
	Status     string `json:"status" gorm:"column:status;size:10"` // 'success', 'failed'
	StartedAt  int64  `json:"started_at" gorm:"column:started_at"`
	FinishedAt int64  `json:"finished_at" gorm:"column:finished_at"`
	Output     string `json:"output" gorm:"column:output;type:jsonb"`
	ErrMessage string `json:"error_message" gorm:"column:error_message"`
	CreatedAt  int64  `json:"created_at" gorm:"column:created_at;autoCreateTime;<-:create"`
	Task       *Task  `json:"task,omitempty" gorm:"foreignKey:TaskID;references:ID;constraint:OnDelete:CASCADE"`
}
