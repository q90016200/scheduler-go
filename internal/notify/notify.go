package notify

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/q90016200/scheduler-go/internal/models"
	"github.com/q90016200/scheduler-go/pkg/logger"
)

// Message 失敗通知的內容，各管道各自決定怎麼呈現
type Message struct {
	TaskID     string
	TaskName   string
	ErrMessage string
	FailedAt   time.Time
}

// Dispatch 依 task 勾選的管道非同步送出通知；單一管道發送失敗只記 log，
// 不重試、不影響其他管道或任務執行本身，比照 logExecution 的錢包哲學
func Dispatch(task *models.Task, msg Message) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if task.NotifyTelegram {
			sendOne(ctx, "telegram", sendTelegram, msg)
		}
		if task.NotifySlack {
			sendOne(ctx, "slack", sendSlack, msg)
		}
	}()
}

func sendOne(ctx context.Context, name string, send func(ctx context.Context, msg Message) error, msg Message) {
	if err := send(ctx, msg); err != nil {
		logger.Log.WithError(err).Errorf("task ID: %s, 發送 %s 失敗通知失敗", msg.TaskID, name)
	}
}

// do 送出請求並判斷結果；必要設定為空時由呼叫端在建構 request 前直接 return nil（不算錯誤）。
// label 只用來組錯誤訊息（例如 "slack webhook"、"telegram API"），維持各管道原本的錯誤文字。
func do(req *http.Request, label string) error {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("%s 回應非 2xx: %d", label, resp.StatusCode)
	}
	return nil
}
