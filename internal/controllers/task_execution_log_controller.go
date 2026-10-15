package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/q90016200/scheduler-go/internal/app"
	"github.com/q90016200/scheduler-go/pkg/response"
)

type TaskExecutionLogController struct {
	services *app.AppServices
}

func NewTaskExecutionLogController(services *app.AppServices) *TaskExecutionLogController {
	return &TaskExecutionLogController{services: services}
}

// ListLogsHandler 列出執行紀錄，可用 task_id 篩選單一任務，limit/offset 分頁
func (ctrl *TaskExecutionLogController) ListLogsHandler(c *gin.Context) {
	taskID := c.Query("task_id")
	taskName := c.Query("task_name")
	groupID := c.Query("group_id")
	status := c.Query("status")
	sortField := c.Query("sort_field")
	sortOrder := c.Query("sort_order")

	limit, offset := pageParams(c)

	logs, total, err := ctrl.services.TaskExecutionLogService.ListLogs(taskID, taskName, groupID, status, sortField, sortOrder, limit, offset)
	if err != nil {
		failErr(c, err, "", "取得執行紀錄失敗")
		return
	}

	response.Success(c, gin.H{"logs": logs, "total": total, "limit": limit, "offset": offset}, "成功獲取執行紀錄")
}

// GetLogHandler 取得單一執行紀錄
func (ctrl *TaskExecutionLogController) GetLogHandler(c *gin.Context) {
	execLog, err := ctrl.services.TaskExecutionLogService.GetLog(c.Param("id"))
	if err != nil {
		failErr(c, err, "執行紀錄不存在", "取得執行紀錄失敗")
		return
	}

	response.Success(c, execLog, "成功獲取執行紀錄")
}

// GetTaskStatsHandler 取得單一任務的執行儀表板小面板（近 30 天成功率/平均耗時、近期失敗次數）
func (ctrl *TaskExecutionLogController) GetTaskStatsHandler(c *gin.Context) {
	stats, err := ctrl.services.TaskExecutionLogService.GetTaskStats(c.Param("id"))
	if err != nil {
		failErr(c, err, "任務不存在", "取得任務統計失敗")
		return
	}

	response.Success(c, stats, "成功獲取任務統計")
}

// GetDashboardStatsHandler 取得全站儀表板統計
func (ctrl *TaskExecutionLogController) GetDashboardStatsHandler(c *gin.Context) {
	stats, err := ctrl.services.TaskExecutionLogService.GetDashboardStats()
	if err != nil {
		failInternal(c, err, "取得儀表板統計失敗")
		return
	}

	response.Success(c, stats, "成功獲取儀表板統計")
}
