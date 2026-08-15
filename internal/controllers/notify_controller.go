package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/q90016200/scheduler-go/config"
	"github.com/q90016200/scheduler-go/pkg/response"
)

type NotifyController struct{}

func NewNotifyController() *NotifyController {
	return &NotifyController{}
}

// GetNotifyConfigHandler 回傳三個通知管道是否已在 .env 設定，只給 bool，絕不回傳實際 token/密碼，
// 供前端在任務表單提醒使用者「勾了但沒設定，通知不會送出」
func (ctrl *NotifyController) GetNotifyConfigHandler(c *gin.Context) {
	nc := config.AppConfig.NotifyConfig
	response.Success(c, gin.H{
		"telegram_configured": nc.TelegramBotToken != "" && nc.TelegramChatID != "",
		"slack_configured":    nc.SlackWebhookURL != "",
		"email_configured":    nc.SMTPHost != "",
	}, "成功獲取通知設定狀態")
}
