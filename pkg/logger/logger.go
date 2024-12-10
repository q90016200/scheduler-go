package logger

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"os"
)

// Logger 全域變數
var Logger *logrus.Logger

// Init 初始化日誌設置
func Init() {
	Logger = logrus.New()

	// 設定輸出至標準輸出
	Logger.Out = os.Stdout

	Logger.SetFormatter(&logrus.JSONFormatter{}) // 使用 JSON 格式，便於存儲和檢索

	// 設定日誌等級
	Logger.SetLevel(logrus.InfoLevel)
}

func WithTrace(c *gin.Context) *logrus.Entry {
	traceID, exists := c.Get("trace_id")
	if !exists {
		traceID = "unknown" // 如果沒有 trace_id，則默認為 unknown
	}

	// 返回包含 trace_id 的 Logrus Entry
	return Logger.WithField("trace_id", traceID)
}

// LogrusAdapter 是一個適配 logrus 的 cron.Logger 實現
type LogrusAdapter struct {
	Logger *logrus.Logger
}

// Info 實現 Info 日志
func (l *LogrusAdapter) Info(msg string, keysAndValues ...interface{}) {
	l.Logger.WithFields(convertFields(keysAndValues...)).Info(msg)
}

// Error 實現 Error 日志
func (l *LogrusAdapter) Error(err error, msg string, keysAndValues ...interface{}) {
	l.Logger.WithFields(convertFields(keysAndValues...)).WithError(err).Error(msg)
}

// convertFields 將鍵值對轉換為 logrus.Fields
func convertFields(keysAndValues ...interface{}) logrus.Fields {
	fields := logrus.Fields{}
	for i := 0; i < len(keysAndValues)-1; i += 2 {
		if key, ok := keysAndValues[i].(string); ok {
			fields[key] = keysAndValues[i+1]
		}
	}
	return fields
}
