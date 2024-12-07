package controllers

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/q90016200/schedule-go/config"
	"github.com/q90016200/schedule-go/internal/app"
	"github.com/q90016200/schedule-go/internal/auth"
	"github.com/q90016200/schedule-go/pkg/response"
	"gorm.io/gorm"
	"net/http"
	"time"
)

type AuthController struct {
	services *app.AppServices
}

func NewAuthController(services *app.AppServices) *AuthController {
	return &AuthController{services: services}
}

func (ctrl *AuthController) RegisterHandler(c *gin.Context) {
	var input struct {
		UserName string `json:"user_name" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Fail(c, http.StatusBadRequest, "請確認使用者名稱和密碼", err)
		return
	}

	if err := ctrl.services.AuthService.Register(input.UserName, input.Password); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			response.Fail(c, http.StatusConflict, "用户名已存在", nil)
			return
		}
		response.Fail(c, http.StatusInternalServerError, "註冊失敗", err)
		return
	}

	response.Success(c, gin.H{}, "註冊成功")
}

func (ctrl *AuthController) LoginHandler(c *gin.Context) {
	var input struct {
		UserName string `json:"user_name" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		response.Fail(c, http.StatusBadRequest, "請確認使用者名稱和密碼", err)
		return
	}

	user, err := ctrl.services.AuthService.Authenticate(input.UserName, input.Password)
	if err != nil {
		response.Fail(c, http.StatusUnauthorized, "使用者名稱或密碼錯誤", err)
		return
	}

	token, err := auth.GenerateToken(user.ID)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "無法生成 Token", err.Error())
		return
	}

	// 设置 HttpOnly cookie
	c.SetCookie("auth_token", token, int(config.AppConfig.TokenExpiry*time.Second), "/", config.AppConfig.ServerDomain, false, true) // `true` 表示 HttpOnly

	response.Success(c, gin.H{
		"token": token,
	}, "登入成功")
	return
}

func (ctrl *AuthController) LogoutHandler(c *gin.Context) {
	fmt.Println("123")
	// 删除 auth_token Cookie
	c.SetCookie("auth_token", "", -1, "/", config.AppConfig.ServerDomain, false, true)

	response.Success(c, gin.H{}, "登出成功")
	return
}
