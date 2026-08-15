package services

import (
	"time"

	"github.com/q90016200/scheduler-go/internal/models"
	"gorm.io/gorm"
)

type TaskExecutionLogService struct {
	db *gorm.DB
}

func NewTaskExecutionLogService(db *gorm.DB) *TaskExecutionLogService {
	return &TaskExecutionLogService{db: db}
}

// listExecutionLogsSortableFields 排序欄位白名單，理由同 task_service.go 的 listTasksSortableFields
var listExecutionLogsSortableFields = map[string]bool{
	"started_at": true,
}

// ListLogs 依 task_id/task_name/group_id/status 篩選（可留空查全部），預設依開始時間新到舊排序，limit/offset 分頁
func (s *TaskExecutionLogService) ListLogs(taskID, taskName, groupID, status, sortField, sortOrder string, limit, offset int) ([]models.TaskExecutionLog, int64, error) {
	var logs []models.TaskExecutionLog
	query := s.db.Model(&models.TaskExecutionLog{})
	if taskID != "" {
		query = query.Where("task_execution_logs.task_id = ?", taskID)
	}
	if status != "" {
		query = query.Where("task_execution_logs.status = ?", status)
	}
	if taskName != "" || groupID != "" {
		query = query.Joins("JOIN tasks ON tasks.id = task_execution_logs.task_id")
		if taskName != "" {
			query = query.Where("tasks.name ILIKE ?", "%"+taskName+"%")
		}
		if groupID != "" {
			query = query.Where("tasks.group_id = ?", groupID)
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	order := "started_at DESC"
	if listExecutionLogsSortableFields[sortField] {
		dir := "ASC"
		if sortOrder == "desc" {
			dir = "DESC"
		}
		order = sortField + " " + dir
	}

	err := query.Preload("Task.Group").Order(order).Limit(limit).Offset(offset).Find(&logs).Error
	return logs, total, err
}

func (s *TaskExecutionLogService) GetLog(id string) (*models.TaskExecutionLog, error) {
	var l models.TaskExecutionLog
	err := s.db.Preload("Task.Group").First(&l, "id = ?", id).Error
	return &l, err
}

// taskStatsQuery 統計只採計排程自動觸發、且是這次觸發最終結果的紀錄，
// 排除立即執行的手動測試與重試中間過程，避免統計失真
func (s *TaskExecutionLogService) taskStatsQuery(taskID string) *gorm.DB {
	return s.db.Model(&models.TaskExecutionLog{}).
		Where("task_id = ? AND source = ? AND is_final_attempt = ?", taskID, models.SOURCE_CRON, true)
}

// GetTaskStats 任務執行儀表板小面板：近 30 天成功率／平均耗時，即時 SQL 聚合，不做 cache
func (s *TaskExecutionLogService) GetTaskStats(taskID string) (*models.TaskStats, error) {
	cutoff := time.Now().AddDate(0, 0, -30).Unix()

	var total30d, success30d int64
	if err := s.taskStatsQuery(taskID).Where("created_at >= ?", cutoff).Count(&total30d).Error; err != nil {
		return nil, err
	}
	if err := s.taskStatsQuery(taskID).
		Where("created_at >= ? AND status = ?", cutoff, models.EXEC_STATUS_SUCCESS).
		Count(&success30d).Error; err != nil {
		return nil, err
	}

	var avgDurationMs float64
	if err := s.taskStatsQuery(taskID).Where("created_at >= ?", cutoff).
		Select("COALESCE(AVG((finished_at - started_at) * 1000), 0)").
		Scan(&avgDurationMs).Error; err != nil {
		return nil, err
	}

	var recentStatuses []string
	if err := s.taskStatsQuery(taskID).Order("started_at DESC").Limit(10).
		Pluck("status", &recentStatuses).Error; err != nil {
		return nil, err
	}
	recentFailCount := 0
	for _, st := range recentStatuses {
		if st == models.EXEC_STATUS_FAILED {
			recentFailCount++
		}
	}

	stats := &models.TaskStats{AvgDurationMs30d: avgDurationMs, RecentFailCount: recentFailCount}
	if total30d > 0 {
		stats.SuccessRate30d = float64(success30d) / float64(total30d) * 100
	}
	return stats, nil
}

// dashboardStatsQuery 統計口徑同 taskStatsQuery，只是不鎖定單一 task_id
func (s *TaskExecutionLogService) dashboardStatsQuery() *gorm.DB {
	return s.db.Model(&models.TaskExecutionLog{}).
		Where("source = ? AND is_final_attempt = ?", models.SOURCE_CRON, true)
}

// GetDashboardStats 全站儀表板：任務啟用/停用數、近 24 小時執行次數與成功率、
// 近 24 小時失敗 Top 5 任務、目前連續失敗中的任務，即時 SQL 聚合，不做 cache
func (s *TaskExecutionLogService) GetDashboardStats() (*models.DashboardStats, error) {
	cutoff := time.Now().Add(-24 * time.Hour).Unix()

	var statusCounts []struct {
		Status string
		N      int64
	}
	if err := s.db.Model(&models.Task{}).Select("status, COUNT(*) AS n").Group("status").Scan(&statusCounts).Error; err != nil {
		return nil, err
	}
	var enabled, disabled int64
	for _, sc := range statusCounts {
		switch sc.Status {
		case models.STATUS_ENABLED:
			enabled = sc.N
		case models.STATUS_DISABLED:
			disabled = sc.N
		}
	}

	var total24h, success24h int64
	if err := s.dashboardStatsQuery().Where("created_at >= ?", cutoff).Count(&total24h).Error; err != nil {
		return nil, err
	}
	if err := s.dashboardStatsQuery().
		Where("created_at >= ? AND status = ?", cutoff, models.EXEC_STATUS_SUCCESS).
		Count(&success24h).Error; err != nil {
		return nil, err
	}

	var topFailing []models.DashboardFailingTask
	if err := s.dashboardStatsQuery().
		Joins("JOIN tasks ON tasks.id = task_execution_logs.task_id").
		Where("task_execution_logs.created_at >= ? AND task_execution_logs.status = ?", cutoff, models.EXEC_STATUS_FAILED).
		Group("tasks.id, tasks.name").
		Select("tasks.id AS task_id, tasks.name AS task_name, COUNT(*) AS fail_count").
		Order("fail_count DESC").
		Limit(5).
		Scan(&topFailing).Error; err != nil {
		return nil, err
	}

	var currentlyFailing []models.DashboardTaskRef
	if err := s.db.Model(&models.Task{}).
		Where("consecutive_failures > 0").
		Select("id AS task_id, name AS task_name").
		Scan(&currentlyFailing).Error; err != nil {
		return nil, err
	}

	stats := &models.DashboardStats{
		TotalTasksEnabled:  enabled,
		TotalTasksDisabled: disabled,
		Executions24h:      total24h,
		TopFailingTasks24h: topFailing,
		CurrentlyFailing:   currentlyFailing,
	}
	if total24h > 0 {
		stats.SuccessRate24h = float64(success24h) / float64(total24h) * 100
	}
	return stats, nil
}
