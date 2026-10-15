package controllers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/q90016200/scheduler-go/config"
	"github.com/q90016200/scheduler-go/internal/app"
	"github.com/q90016200/scheduler-go/internal/auth"
	"github.com/q90016200/scheduler-go/internal/models"
	"github.com/q90016200/scheduler-go/internal/services"
	"github.com/q90016200/scheduler-go/pkg/response"
	"gorm.io/gorm"
)

type AuthController struct {
	services *app.AppServices
}

func NewAuthController(services *app.AppServices) *AuthController {
	return &AuthController{services: services}
}

func (ctrl *AuthController) RegisterHandler(c *gin.Context) {
	var input struct {
		UserName string `json:"user_name" validate:"required"`
		Password string `json:"password" validate:"required"`
	}
	if !bindJSON(c, &input) {
		return
	}

	_, err := ctrl.services.UserService.CreateUser(models.CreateUserReq{UserName: input.UserName, Password: input.Password})
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			response.Fail(c, http.StatusConflict, "用户名已存在", nil)
			return
		}
		failInternal(c, err, "註冊失敗")
		return
	}

	response.Success(c, gin.H{}, "註冊成功")
}

func (ctrl *AuthController) LoginHandler(c *gin.Context) {
	var input struct {
		UserName string `json:"user_name" validate:"required"`
		Password string `json:"password" validate:"required"`
	}

	if !bindJSON(c, &input) {
		return
	}

	user, err := ctrl.services.AuthService.Authenticate(input.UserName, input.Password)
	if err != nil {
		if errors.Is(err, services.ErrUserDisabled) {
			response.Fail(c, http.StatusUnauthorized, "帳號已停用", nil)
			return
		}
		if errors.Is(err, services.ErrInvalidCredentials) {
			response.Fail(c, http.StatusUnauthorized, "使用者名稱或密碼錯誤", nil)
			return
		}
		failInternal(c, err, "登入失敗")
		return
	}

	token, err := auth.GenerateToken(user.ID)
	if err != nil {
		failInternal(c, err, "無法生成 Token")
		return
	}

	// 设置 HttpOnly cookie
	c.SetCookie("auth_token", token, config.AppConfig.TokenExpirySeconds, "/", config.AppConfig.ServerDomain, false, true) // `true` 表示 HttpOnly

	response.Success(c, gin.H{
		"token": token,
	}, "登入成功")
	return
}

func (ctrl *AuthController) LogoutHandler(c *gin.Context) {
	// 删除 auth_token Cookie
	c.SetCookie("auth_token", "", -1, "/", config.AppConfig.ServerDomain, false, true)

	response.Success(c, gin.H{}, "登出成功")
	return
}

// MeHandler 取得目前登入使用者
func (ctrl *AuthController) MeHandler(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.Fail(c, http.StatusUnauthorized, "未登入", nil)
		return
	}

	user, err := ctrl.services.UserService.GetUser(userID.(string))
	if err != nil {
		failErr(c, err, "使用者不存在", "取得使用者失敗")
		return
	}

	response.Success(c, user, "成功")
}
