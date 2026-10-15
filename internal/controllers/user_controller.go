package controllers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/q90016200/scheduler-go/internal/app"
	"github.com/q90016200/scheduler-go/internal/models"
	"github.com/q90016200/scheduler-go/pkg/response"
	"gorm.io/gorm"
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

	limit, offset := pageParams(c)

	users, total, err := ctrl.services.UserService.ListUsers(name, limit, offset)
	if err != nil {
		failErr(c, err, "", "取得使用者清單失敗")
		return
	}

	response.Success(c, gin.H{"users": users, "total": total, "limit": limit, "offset": offset}, "成功獲取使用者清單")
}

// GetUserHandler 取得單一使用者
func (ctrl *UserController) GetUserHandler(c *gin.Context) {
	user, err := ctrl.services.UserService.GetUser(c.Param("id"))
	if err != nil {
		failErr(c, err, "使用者不存在", "取得使用者失敗")
		return
	}

	response.Success(c, user, "成功獲取使用者")
}

// CreateUserHandler 建立使用者
func (ctrl *UserController) CreateUserHandler(c *gin.Context) {

	var req models.CreateUserReq
	if !bindJSON(c, &req) {
		return
	}

	user, err := ctrl.services.UserService.CreateUser(req)
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			response.Fail(c, http.StatusConflict, "使用者名稱已存在", nil)
			return
		}
		failInternal(c, err, "建立使用者失敗")
		return
	}

	response.Success(c, user, "使用者已建立")
}

// UpdateUserHandler 更新使用者
func (ctrl *UserController) UpdateUserHandler(c *gin.Context) {

	var req models.UpdateUserReq
	if !bindJSON(c, &req) {
		return
	}

	user, err := ctrl.services.UserService.UpdateUser(c.Param("id"), req)
	if err != nil {
		failErr(c, err, "使用者不存在", "更新使用者失敗")
		return
	}

	response.Success(c, user, "使用者已更新")
}

// UpdateUserStatusHandler 切換使用者啟用/停用狀態
func (ctrl *UserController) UpdateUserStatusHandler(c *gin.Context) {

	var req models.UpdateUserStatusReq
	if !bindJSON(c, &req) {
		return
	}

	if err := ctrl.services.UserService.UpdateUserStatus(c.Param("id"), req.Status); err != nil {
		failErr(c, err, "使用者不存在", "更新使用者狀態失敗")
		return
	}

	response.Success(c, gin.H{}, "使用者狀態已更新")
}

// DeleteUserHandler 刪除使用者
func (ctrl *UserController) DeleteUserHandler(c *gin.Context) {
	if err := ctrl.services.UserService.DeleteUser(c.Param("id")); err != nil {
		failErr(c, err, "使用者不存在", "刪除使用者失敗")
		return
	}

	response.Success(c, gin.H{}, "使用者已刪除")
}
