package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/q90016200/scheduler-go/config"
)

func sendSlack(ctx context.Context, msg Message) error {
	cfg := config.AppConfig.NotifyConfig
	if !cfg.SlackConfigured() {
		return nil
	}
	webhookURL := cfg.SlackWebhookURL

	text := fmt.Sprintf("任務失敗：*%s*\n時間：%s\n錯誤：%s", msg.TaskName, msg.FailedAt.Format("2006-01-02 15:04:05"), msg.ErrMessage)
	body, err := json.Marshal(map[string]string{"text": text})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	return do(req, "slack webhook")
}
