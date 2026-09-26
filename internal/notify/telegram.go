package notify

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/q90016200/scheduler-go/config"
)

func sendTelegram(ctx context.Context, msg Message) error {
	cfg := config.AppConfig.NotifyConfig
	if cfg.TelegramBotToken == "" || cfg.TelegramChatID == "" {
		return nil
	}

	text := fmt.Sprintf("任務失敗：%s\n時間：%s\n錯誤：%s", msg.TaskName, msg.FailedAt.Format("2006-01-02 15:04:05"), msg.ErrMessage)
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", cfg.TelegramBotToken)

	form := url.Values{}
	form.Set("chat_id", cfg.TelegramChatID)
	form.Set("text", text)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, nil)
	if err != nil {
		return err
	}
	req.URL.RawQuery = form.Encode()

	return do(req, "telegram API")
}
