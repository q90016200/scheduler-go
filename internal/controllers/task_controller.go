package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/q90016200/schedule-go/config"
	"github.com/q90016200/schedule-go/internal/app"
	"github.com/q90016200/schedule-go/pkg/logger"
	"github.com/q90016200/schedule-go/pkg/response"
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
	// 建立任務的邏輯
	c.JSON(http.StatusCreated, gin.H{"task": "任務已建立"})
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
