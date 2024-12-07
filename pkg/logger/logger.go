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
