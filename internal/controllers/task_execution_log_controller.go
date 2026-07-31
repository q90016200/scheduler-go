package controllers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/q90016200/scheduler-go/internal/app"
	"github.com/q90016200/scheduler-go/pkg/response"
	"gorm.io/gorm"
	"net/http"
	"strconv"
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

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if offset < 0 {
		offset = 0
	}

	logs, total, err := ctrl.services.TaskExecutionLogService.ListLogs(taskID, taskName, groupID, limit, offset)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "取得執行紀錄失敗", err.Error())
		return
	}

	response.Success(c, gin.H{"logs": logs, "total": total, "limit": limit, "offset": offset}, "成功獲取執行紀錄")
}

// GetLogHandler 取得單一執行紀錄
func (ctrl *TaskExecutionLogController) GetLogHandler(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Fail(c, http.StatusBadRequest, "id is empty", nil)
		return
	}

	log, err := ctrl.services.TaskExecutionLogService.GetLog(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, http.StatusNotFound, "執行紀錄不存在", nil)
			return
		}
		response.Fail(c, http.StatusInternalServerError, "取得執行紀錄失敗", err.Error())
		return
	}

	response.Success(c, log, "成功獲取執行紀錄")
}
