package controllers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/q90016200/scheduler-go/internal/app"
	"github.com/q90016200/scheduler-go/internal/models"
	"github.com/q90016200/scheduler-go/internal/scheduler"
	"github.com/q90016200/scheduler-go/pkg/logger"
	"github.com/q90016200/scheduler-go/pkg/response"
	"github.com/q90016200/scheduler-go/pkg/utils"
	"gorm.io/gorm"
	"io"
	"net/http"
)

type TaskController struct {
	services *app.AppServices
}

func NewTaskController(services *app.AppServices) *TaskController {
	return &TaskController{services: services}
}

func (ctrl *TaskController) ListTasksHandler(c *gin.Context) {
	// 提取查詢參數
	status := c.Query("status")    // 篩選條件：status
	groupID := c.Query("group_id") // 篩選條件：group_id
	name := c.Query("name")        // 篩選條件：name（模糊比對）

	sortField := c.Query("sort_field") // 排序字段
	sortOrder := c.Query("sort_order") // 排序方式（asc 或 desc）

	limit, offset := pageParams(c)

	// 調用 Service
	tasks, total, err := ctrl.services.TaskServices.ListTasks(status, groupID, name, sortField, sortOrder, limit, offset)
	if err != nil {
		logger.WithTrace(c).Error(err.Error())
		response.Fail(c, http.StatusInternalServerError, "取得任務清單失敗", nil)
		return
	}

	// 取得任務清單的邏輯
	response.Success(c, gin.H{"tasks": tasks, "total": total, "limit": limit, "offset": offset}, "成功獲取任務清單")
}

// GetTaskHandler 取得單一任務
func (ctrl *TaskController) GetTaskHandler(c *gin.Context) {
	task, err := ctrl.services.TaskServices.GetTask(c.Param("id"))
	if err != nil {
		failErr(c, err, "任務不存在", "取得任務失敗")
		return
	}

	response.Success(c, task, "成功獲取任務")
}

// CreateTaskHandler 建立任務
func (ctrl *TaskController) CreateTaskHandler(c *gin.Context) {
	// 獲取帶有 trace_id 的 logger
	log := logger.WithTrace(c)

	// 解析並綁定請求數據到 CreateTaskReq 結構體
	var req models.CreateTaskReq
	if !bindJSON(c, &req) {
		return
	}

	// 當請求參數 status 未傳入時，預設為不啟動
	if req.Status == "" {
		req.Status = models.STATUS_DISABLED
	}

	task, err := ctrl.services.TaskServices.CreateTask(req)
	if err != nil {
		log.Error(err.Error())
		response.Fail(c, http.StatusInternalServerError, "建立任務失敗", nil)
		return
	}

	response.Success(c, task, "任務已建立")
}

// UpdateTaskHandler 更新任務
func (ctrl *TaskController) UpdateTaskHandler(c *gin.Context) {
	// 獲取帶有 trace_id 的 logger

	var req models.UpdateTaskReq
	if !bindJSON(c, &req) {
		return
	}

	task, err := ctrl.services.TaskServices.UpdateTask(c.Param("id"), req)
	if err != nil {
		failErr(c, err, "任務不存在", "更新任務失敗")
		return
	}

	response.Success(c, task, "任務已更新")
}

// UpdateTaskStatusHandler 切換任務啟用/停用狀態
func (ctrl *TaskController) UpdateTaskStatusHandler(c *gin.Context) {
	var req models.UpdateTaskStatusReq
	if !bindJSON(c, &req) {
		return
	}

	task, err := ctrl.services.TaskServices.UpdateTaskStatus(c.Param("id"), req.Status, req.UserId)
	if err != nil {
		failErr(c, err, "任務不存在", "更新任務狀態失敗")
		return
	}

	response.Success(c, task, "任務狀態已更新")
}

// RunTaskHandler 立即執行任務，可帶 repeat/interval_ms 指定重複次數與呼叫間隔；body 可省略，預設執行 1 次
func (ctrl *TaskController) RunTaskHandler(c *gin.Context) {
	log := logger.WithTrace(c)

	id := c.Param("id")

	var req models.RunTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		log.Info(err.Error())
		response.Fail(c, http.StatusBadRequest, "參數驗證失敗", utils.FormatValidationErrors(err))
		return
	}
	if err := utils.MyValidator.Struct(req); err != nil {
		log.Info(err.Error())
		response.Fail(c, http.StatusBadRequest, "參數驗證失敗", utils.FormatValidationErrors(err))
		return
	}
	if req.Repeat == 0 {
		req.Repeat = 1
	}
	if req.IntervalMs == 0 {
		req.IntervalMs = 1000
	}

	task, err := ctrl.services.TaskServices.RunTaskNow(id, req.Repeat, req.IntervalMs)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, http.StatusNotFound, "任務不存在", nil)
			return
		}
		if errors.Is(err, scheduler.ErrBatchAlreadyRunning) {
			response.Fail(c, http.StatusConflict, "該任務已有執行中的批次", nil)
			return
		}
		log.Error(err.Error())
		response.Fail(c, http.StatusInternalServerError, "執行任務失敗", nil)
		return
	}

	response.Success(c, gin.H{"task_id": task.ID, "repeat": req.Repeat, "interval_ms": req.IntervalMs}, "已觸發執行")
}

// CancelTaskRunHandler 取消任務目前進行中的「立即執行」重複批次
func (ctrl *TaskController) CancelTaskRunHandler(c *gin.Context) {
	if err := ctrl.services.TaskServices.CancelRunNow(c.Param("id")); err != nil {
		if errors.Is(err, scheduler.ErrBatchNotFound) {
			response.Fail(c, http.StatusNotFound, "找不到該任務進行中的批次", nil)
			return
		}
		logger.WithTrace(c).Error(err.Error())
		response.Fail(c, http.StatusInternalServerError, "取消執行失敗", nil)
		return
	}

	response.Success(c, nil, "已取消")
}

// DeleteTaskHandler 刪除任務
func (ctrl *TaskController) DeleteTaskHandler(c *gin.Context) {
	if err := ctrl.services.TaskServices.DeleteTask(c.Param("id")); err != nil {
		failErr(c, err, "任務不存在", "刪除任務失敗")
		return
	}
	response.Success(c, nil, "任務已刪除")
}

// ListTaskGroupsHandler 列出所有任務群組
func (ctrl *TaskController) ListTaskGroupsHandler(c *gin.Context) {
	status := c.Query("status")
	name := c.Query("name")
	sortField := c.Query("sort_field")
	sortOrder := c.Query("sort_order")

	limit, offset := pageParams(c)

	groups, total, err := ctrl.services.TaskServices.ListTaskGroups(status, name, sortField, sortOrder, limit, offset)
	if err != nil {
		logger.WithTrace(c).Error(err.Error())
		response.Fail(c, http.StatusInternalServerError, "取得任務群組清單失敗", nil)
		return
	}
	response.Success(c, gin.H{"groups": groups, "total": total, "limit": limit, "offset": offset}, "成功獲取任務群組清單")
}

// GetTaskGroupHandler 取得單一任務群組
func (ctrl *TaskController) GetTaskGroupHandler(c *gin.Context) {
	group, err := ctrl.services.TaskServices.GetTaskGroup(c.Param("id"))
	if err != nil {
		failErr(c, err, "任務群組不存在", "取得任務群組失敗")
		return
	}

	response.Success(c, group, "成功獲取任務群組")
}

// CreateTaskGroupHandler 建立任務群組
func (ctrl *TaskController) CreateTaskGroupHandler(c *gin.Context) {
	// 獲取帶有 trace_id 的 logger
	log := logger.WithTrace(c)

	// 解析並綁定請求數據到 CreateTaskGroupReq 結構體
	var req models.CreateTaskGroupReq
	if !bindJSON(c, &req) {
		return
	}

	// 當請求參數 status 未傳入時，預設為不啟動
	if req.Status == "" {
		req.Status = models.STATUS_DISABLED
	}

	task, err := ctrl.services.TaskServices.CreateTaskGroup(req)
	if err != nil {
		log.Error(err.Error())
		response.Fail(c, http.StatusInternalServerError, "建立任務群組失敗", nil)
		return
	}

	response.Success(c, task, "任務群組已建立")
}

// UpdateTaskGroupHandler 更新任務群組
func (ctrl *TaskController) UpdateTaskGroupHandler(c *gin.Context) {
	// 獲取帶有 trace_id 的 logger

	var req models.UpdateTaskGroupReq
	if !bindJSON(c, &req) {
		return
	}

	task, err := ctrl.services.TaskServices.UpdateTaskGroup(c.Param("id"), req)
	if err != nil {
		failErr(c, err, "任務群組不存在", "更新任務群組失敗")
		return
	}

	response.Success(c, task, "任務群組已更新")
}

// DeleteTaskGroupHandler 刪除任務群組
func (ctrl *TaskController) DeleteTaskGroupHandler(c *gin.Context) {
	deactivated, err := ctrl.services.TaskServices.DeleteTaskGroup(c.Param("id"))
	if err != nil {
		failErr(c, err, "任務群組不存在", "刪除任務群組失敗")
		return
	}

	deactivatedTasks := make([]gin.H, 0, len(deactivated))
	for _, task := range deactivated {
		deactivatedTasks = append(deactivatedTasks, gin.H{"id": task.ID, "name": task.Name})
	}
	response.Success(c, gin.H{"deactivated_tasks": deactivatedTasks}, "任務群組已刪除")
}
