package notify

import (
	"context"
	"fmt"
	"net/smtp"

	"github.com/q90016200/scheduler-go/config"
)

type emailChannel struct{}

// ponytail: net/smtp.SendMail 不吃 context，逾時交給 SMTP server 端連線行為決定；
// 量大到需要真的取消/逾時控制時再換成自建 dialer
func (emailChannel) Send(_ context.Context, msg Message) error {
	cfg := config.AppConfig.NotifyConfig
	if cfg.SMTPHost == "" || cfg.SMTPFrom == "" || cfg.SMTPTo == "" {
		return nil
	}

	addr := fmt.Sprintf("%s:%s", cfg.SMTPHost, cfg.SMTPPort)
	subject := fmt.Sprintf("[排程任務失敗] %s", msg.TaskName)
	body := fmt.Sprintf("任務失敗：%s\r\n時間：%s\r\n錯誤：%s\r\n",
		msg.TaskName, msg.FailedAt.Format("2006-01-02 15:04:05"), msg.ErrMessage)
	mail := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", cfg.SMTPFrom, cfg.SMTPTo, subject, body)

	var auth smtp.Auth
	if cfg.SMTPUser != "" {
		auth = smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPassword, cfg.SMTPHost)
	}

	return smtp.SendMail(addr, auth, cfg.SMTPFrom, []string{cfg.SMTPTo}, []byte(mail))
}
