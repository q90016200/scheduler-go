package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/q90016200/schedule-go/config"
	"github.com/q90016200/schedule-go/internal/app"
	"github.com/q90016200/schedule-go/internal/models"
	"github.com/q90016200/schedule-go/pkg/logger"
	"github.com/q90016200/schedule-go/pkg/response"
	"github.com/q90016200/schedule-go/pkg/utils"
	"net/http"
)

type TaskController struct {
	services *app.AppServices
}

func NewTaskController(services *app.AppServices) *TaskController {
	return &TaskController{services: services}
}

func (ctrl *TaskController) ListTasksHandler(c *gin.Context) {
	// 取得 Trace ID
	traceID, exists := c.Get("TraceID")
	if !exists {
		traceID = "unknown"
	}

	if !config.AppConfig.TaskEnabled {
		logger.Logger.Warn("任務系統已停用")
		c.JSON(http.StatusForbidden, gin.H{"error": "任務系統已停用"})
		return
	}

	tasks := []string{"task1", "task2", "task3"}
	logger.Logger.Info("成功獲取任務清單")

	logger.Logger.Info("Processing request in SomeHandler, Trace ID: ", traceID)

	// 取得任務清單的邏輯
	response.Success(c, gin.H{"tasks": tasks}, "成功獲取任務清單")
}

// CreateTaskHandler 建立任務
func (ctrl *TaskController) CreateTaskHandler(c *gin.Context) {
	// 解析並綁定請求數據到 CreateTaskReq 結構體
	var req models.CreateTaskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		// 解析 JSON 時的錯誤
		response.Fail(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	if err := utils.MyValidator.Struct(req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// 當請求參數 status 未傳入時，預設為不啟動
	if req.Status == "" {
		req.Status = models.TASK_STATUS_INACTIVE
	}

	task, err := ctrl.services.TaskServices.CreateTask(req)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "建立任務失敗", err.Error())
		return
	}

	response.Success(c, task, "任務已建立")
	return
}

// UpdateTaskHandler 更新任務
func (ctrl *TaskController) UpdateTaskHandler(c *gin.Context) {
	// 更新指定任務的邏輯
	c.JSON(http.StatusOK, gin.H{"task": "任務已更新"})
}

// DeleteTaskHandler 刪除任務
func (ctrl *TaskController) DeleteTaskHandler(c *gin.Context) {
	// 刪除指定任務的邏輯
	c.JSON(http.StatusOK, gin.H{"message": "任務已刪除"})
}
