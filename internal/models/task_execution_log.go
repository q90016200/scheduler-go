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

	Source         string `json:"source" gorm:"column:source;size:10"`          // "cron" / "manual"
	AttemptNumber  int    `json:"attempt_number" gorm:"column:attempt_number"`  // 第幾次嘗試，第一次是 1
	IsFinalAttempt bool   `json:"is_final_attempt" gorm:"column:is_final_attempt"` // 這次觸發（含所有重試）的最後一筆記錄
}

const (
	SOURCE_CRON   = "cron"
	SOURCE_MANUAL = "manual"
)

// TaskStats 任務執行儀表板小面板；只採計 source=cron AND is_final_attempt=true 的紀錄，
// 排除立即執行的手動測試與重試中間過程，避免統計失真
type TaskStats struct {
	SuccessRate30d   float64 `json:"success_rate_30d"`   // 近 30 天成功率（百分比），無資料時為 0
	AvgDurationMs30d float64 `json:"avg_duration_ms_30d"` // 近 30 天平均耗時（毫秒）
	RecentFailCount  int     `json:"recent_fail_count"`   // 最近 10 次中失敗幾次
}
