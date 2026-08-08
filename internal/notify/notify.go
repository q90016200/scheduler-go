package notify

import (
	"context"
	"time"

	"github.com/q90016200/scheduler-go/internal/models"
	"github.com/q90016200/scheduler-go/pkg/logger"
)

// Message 失敗通知的內容，三個管道各自決定怎麼呈現
type Message struct {
	TaskID     string
	TaskName   string
	ErrMessage string
	FailedAt   time.Time
}

// Channel 一個通知管道；必要設定為空時 Send 直接跳過（不算錯誤），由呼叫端決定要不要送
type Channel interface {
	Send(ctx context.Context, msg Message) error
}

// Dispatch 依 task 勾選的管道非同步送出通知；單一管道發送失敗只記 log，
// 不重試、不影響其他管道或任務執行本身，比照 logExecution 的錢包哲學
func Dispatch(task *models.Task, msg Message) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if task.NotifyTelegram {
			sendOne(ctx, "telegram", telegramChannel{}, msg)
		}
		if task.NotifySlack {
			sendOne(ctx, "slack", slackChannel{}, msg)
		}
		if task.NotifyEmail {
			sendOne(ctx, "email", emailChannel{}, msg)
		}
	}()
}

func sendOne(ctx context.Context, name string, ch Channel, msg Message) {
	if err := ch.Send(ctx, msg); err != nil {
		logger.Log.WithError(err).Errorf("task ID: %s, 發送 %s 失敗通知失敗", msg.TaskID, name)
	}
}
