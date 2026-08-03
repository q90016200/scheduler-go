package controllers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/q90016200/scheduler-go/internal/app"
	"github.com/q90016200/scheduler-go/internal/models"
	"github.com/q90016200/scheduler-go/pkg/logger"
	"github.com/q90016200/scheduler-go/pkg/response"
	"github.com/q90016200/scheduler-go/pkg/utils"
	"gorm.io/gorm"
	"net/http"
	"strconv"
)

type UserController struct {
	services *app.AppServices
}

func NewUserController(services *app.AppServices) *UserController {
	return &UserController{services: services}
}

// ListUsersHandler 列出使用者，可用 name 模糊篩選，limit/offset 分頁
func (ctrl *UserController) ListUsersHandler(c *gin.Context) {
	name := c.Query("name")

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if offset < 0 {
		offset = 0
	}

	users, total, err := ctrl.services.UserService.ListUsers(name, limit, offset)
	if err != nil {
		logger.WithTrace(c).Error(err.Error())
		response.Fail(c, http.StatusInternalServerError, "取得使用者清單失敗", nil)
		return
	}

	response.Success(c, gin.H{"users": users, "total": total, "limit": limit, "offset": offset}, "成功獲取使用者清單")
}

// GetUserHandler 取得單一使用者
func (ctrl *UserController) GetUserHandler(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Fail(c, http.StatusBadRequest, "id is empty", nil)
		return
	}

	user, err := ctrl.services.UserService.GetUser(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, http.StatusNotFound, "使用者不存在", nil)
			return
		}
		logger.WithTrace(c).Error(err.Error())
		response.Fail(c, http.StatusInternalServerError, "取得使用者失敗", nil)
		return
	}

	response.Success(c, user, "成功獲取使用者")
}

// CreateUserHandler 建立使用者
func (ctrl *UserController) CreateUserHandler(c *gin.Context) {
	log := logger.WithTrace(c)

	var req models.CreateUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error(err.Error())
		response.Fail(c, http.StatusBadRequest, "參數驗證失敗", utils.FormatValidationErrors(err))
		return
	}

	if err := utils.MyValidator.Struct(req); err != nil {
		log.Error(err.Error())
		response.Fail(c, http.StatusBadRequest, "參數驗證失敗", utils.FormatValidationErrors(err))
		return
	}

	user, err := ctrl.services.UserService.CreateUser(req)
	if err != nil {
		log.Error(err.Error())
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			response.Fail(c, http.StatusConflict, "使用者名稱已存在", nil)
			return
		}
		response.Fail(c, http.StatusInternalServerError, "建立使用者失敗", nil)
		return
	}

	response.Success(c, user, "使用者已建立")
}

// UpdateUserHandler 更新使用者
func (ctrl *UserController) UpdateUserHandler(c *gin.Context) {
	log := logger.WithTrace(c)

	id := c.Param("id")
	if id == "" {
		response.Fail(c, http.StatusBadRequest, "id is empty", nil)
		return
	}

	var req models.UpdateUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error(err.Error())
		response.Fail(c, http.StatusBadRequest, "參數驗證失敗", utils.FormatValidationErrors(err))
		return
	}

	if err := utils.MyValidator.Struct(req); err != nil {
		log.Error(err.Error())
		response.Fail(c, http.StatusBadRequest, "參數驗證失敗", utils.FormatValidationErrors(err))
		return
	}

	user, err := ctrl.services.UserService.UpdateUser(id, req)
	if err != nil {
		log.Error(err.Error())
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, http.StatusNotFound, "使用者不存在", nil)
			return
		}
		response.Fail(c, http.StatusInternalServerError, "更新使用者失敗", nil)
		return
	}

	response.Success(c, user, "使用者已更新")
}

// UpdateUserStatusHandler 切換使用者啟用/停用狀態
func (ctrl *UserController) UpdateUserStatusHandler(c *gin.Context) {
	log := logger.WithTrace(c)

	id := c.Param("id")
	if id == "" {
		response.Fail(c, http.StatusBadRequest, "id is empty", nil)
		return
	}

	var req models.UpdateUserStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error(err.Error())
		response.Fail(c, http.StatusBadRequest, "參數驗證失敗", utils.FormatValidationErrors(err))
		return
	}

	if err := utils.MyValidator.Struct(req); err != nil {
		log.Error(err.Error())
		response.Fail(c, http.StatusBadRequest, "參數驗證失敗", utils.FormatValidationErrors(err))
		return
	}

	if err := ctrl.services.UserService.UpdateUserStatus(id, req.Status); err != nil {
		log.Error(err.Error())
		response.Fail(c, http.StatusInternalServerError, "更新使用者狀態失敗", nil)
		return
	}

	response.Success(c, gin.H{}, "使用者狀態已更新")
}

// DeleteUserHandler 刪除使用者
func (ctrl *UserController) DeleteUserHandler(c *gin.Context) {
	log := logger.WithTrace(c)

	id := c.Param("id")
	if id == "" {
		response.Fail(c, http.StatusBadRequest, "id is empty", nil)
		return
	}

	if err := ctrl.services.UserService.DeleteUser(id); err != nil {
		log.Error(err.Error())
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, http.StatusNotFound, "使用者不存在", nil)
			return
		}
		response.Fail(c, http.StatusInternalServerError, "刪除使用者失敗", nil)
		return
	}

	response.Success(c, gin.H{}, "使用者已刪除")
}
