package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/q90016200/scheduler-go/internal/app"
	"github.com/q90016200/scheduler-go/internal/models"
	"github.com/q90016200/scheduler-go/pkg/logger"
	"github.com/q90016200/scheduler-go/pkg/response"
	"github.com/q90016200/scheduler-go/pkg/utils"
	"net/http"
)

type TaskController struct {
	services *app.AppServices
}

func NewTaskController(services *app.AppServices) *TaskController {
	return &TaskController{services: services}
}

func (ctrl *TaskController) ListTasksHandler(c *gin.Context) {
	//if !config.AppConfig.TaskEnabled {
	//	logger.Logger.Warn("任務系統已停用")
	//	c.JSON(http.StatusForbidden, gin.H{"error": "任務系統已停用"})
	//	return
	//}

	// 提取查詢參數
	status := c.Query("status") // 篩選條件：status

	sortField := c.Query("sort_field") // 排序字段
	sortOrder := c.Query("sort_order") // 排序方式（asc 或 desc）

	// 調用 Service
	tasks, err := ctrl.services.TaskServices.ListTasks(status, sortField, sortOrder)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "取得任務清單失敗", err.Error())
		return
	}

	// 取得任務清單的邏輯
	response.Success(c, gin.H{"tasks": tasks}, "成功獲取任務清單")
}

// CreateTaskHandler 建立任務
func (ctrl *TaskController) CreateTaskHandler(c *gin.Context) {
	// 獲取帶有 trace_id 的 logger
	log := logger.WithTrace(c)

	// 解析並綁定請求數據到 CreateTaskReq 結構體
	var req models.CreateTaskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error(err.Error())
		// 解析 JSON 時的錯誤
		response.Fail(c, http.StatusBadRequest, "參數驗證失敗", utils.FormatValidationErrors(err))
		return
	}

	if err := utils.MyValidator.Struct(req); err != nil {
		log.Error(err.Error())
		response.Fail(c, http.StatusBadRequest, "參數驗證失敗", utils.FormatValidationErrors(err))
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
	// 獲取帶有 trace_id 的 logger
	log := logger.WithTrace(c)

	id := c.Param("id")
	if id == "" {
		errMsg := "id is empty"
		log.Info(errMsg)
		response.Fail(c, http.StatusBadRequest, errMsg, nil)
		return
	}

	var req models.UpdateTaskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		// 解析 JSON 時的錯誤
		log.Info(err.Error())
		response.Fail(c, http.StatusBadRequest, "參數驗證失敗", utils.FormatValidationErrors(err))
		return
	}

	if err := utils.MyValidator.Struct(req); err != nil {
		log.Info(err.Error())
		response.Fail(c, http.StatusBadRequest, "參數驗證失敗", utils.FormatValidationErrors(err))
		return
	}

	task, err := ctrl.services.TaskServices.UpdateTask(id, req)
	if err != nil {
		log.Info(err.Error())
		response.Fail(c, http.StatusInternalServerError, "更新任務失敗", err.Error())
		return
	}

	response.Success(c, task, "任務已更新")
	//c.JSON(http.StatusOK, gin.H{"task": "任務已更新"})
}

// DeleteTaskHandler 刪除任務
func (ctrl *TaskController) DeleteTaskHandler(c *gin.Context) {
	// 獲取帶有 trace_id 的 logger
	log := logger.WithTrace(c)

	id := c.Param("id")
	if id == "" {
		errMsg := "id is empty"
		log.Info(errMsg)
		response.Fail(c, http.StatusBadRequest, errMsg, nil)
		return
	}

	if err := ctrl.services.TaskServices.DeleteTask(id); err != nil {
		log.Info(err.Error())
		response.Fail(c, http.StatusInternalServerError, "刪除任務失敗", err.Error())
		return
	}
	response.Success(c, nil, "任務已刪除")
	//c.JSON(http.StatusOK, gin.H{"message": "任務已刪除"})
}

// CreateTaskGroupHandler 建立任務群組
func (ctrl *TaskController) CreateTaskGroupHandler(c *gin.Context) {
	// 獲取帶有 trace_id 的 logger
	log := logger.WithTrace(c)

	// 解析並綁定請求數據到 CreateTaskGroupReq 結構體
	var req models.CreateTaskGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error(err.Error())
		// 解析 JSON 時的錯誤
		response.Fail(c, http.StatusBadRequest, "參數驗證失敗", utils.FormatValidationErrors(err))
		return
	}

	if err := utils.MyValidator.Struct(req); err != nil {
		log.Error(err.Error())
		response.Fail(c, http.StatusBadRequest, "參數驗證失敗", utils.FormatValidationErrors(err))
		return
	}

	// 當請求參數 status 未傳入時，預設為不啟動
	if req.Status == "" {
		req.Status = models.TASK_STATUS_INACTIVE
	}

	task, err := ctrl.services.TaskServices.CreateTaskGroup(req)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "建立任務群組失敗", err.Error())
		return
	}

	response.Success(c, task, "任務群組已建立")
	return
}

// UpdateTaskGroupHandler 更新任務群組
func (ctrl *TaskController) UpdateTaskGroupHandler(c *gin.Context) {
	// 獲取帶有 trace_id 的 logger
	log := logger.WithTrace(c)

	id := c.Param("id")
	if id == "" {
		errMsg := "id is empty"
		log.Info(errMsg)
		response.Fail(c, http.StatusBadRequest, errMsg, nil)
		return
	}

	var req models.UpdateTaskGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		// 解析 JSON 時的錯誤
		log.Info(err.Error())
		response.Fail(c, http.StatusBadRequest, "參數驗證失敗", utils.FormatValidationErrors(err))
		return
	}

	if err := utils.MyValidator.Struct(req); err != nil {
		log.Info(err.Error())
		response.Fail(c, http.StatusBadRequest, "參數驗證失敗", utils.FormatValidationErrors(err))
		return
	}

	task, err := ctrl.services.TaskServices.UpdateTaskGroup(id, req)
	if err != nil {
		log.Info(err.Error())
		response.Fail(c, http.StatusInternalServerError, "更新任務群組失敗", err.Error())
		return
	}

	response.Success(c, task, "任務群組已更新")
}

// DeleteTaskGroupHandler 刪除任務群組
func (ctrl *TaskController) DeleteTaskGroupHandler(c *gin.Context) {
	// 獲取帶有 trace_id 的 logger
	log := logger.WithTrace(c)

	id := c.Param("id")
	if id == "" {
		errMsg := "id is empty"
		log.Info(errMsg)
		response.Fail(c, http.StatusBadRequest, errMsg, nil)
		return
	}

	if err := ctrl.services.TaskServices.DeleteTaskGroup(id); err != nil {
		log.Info(err.Error())
		response.Fail(c, http.StatusInternalServerError, "刪除任務群組失敗", err.Error())
		return
	}
	response.Success(c, nil, "任務群組已刪除")
	//c.JSON(http.StatusOK, gin.H{"message": "任務群組已刪除"})
}
