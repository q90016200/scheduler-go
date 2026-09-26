package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/q90016200/scheduler-go/pkg/logger"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
	"time"
)

func OpenTracingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 優先沿用上游傳入的 X-Request-ID，方便跨服務追蹤；沒有才自行生成
		traceID := c.GetHeader("X-Request-ID")
		if traceID == "" {
			traceID = xid.New().String()
		}

		// 將 Trace ID 加入到 Context 中，並回寫 response header 供呼叫方對照
		c.Set("trace_id", traceID)
		c.Header("X-Request-ID", traceID)

		start := time.Now()

		// 處理請求
		c.Next()

		// 請求開始與結束合併成一行，避免每個請求印兩行 Info（自動刷新等高頻輪詢會讓 log 量翻倍）
		logger.WithTrace(c).WithFields(logrus.Fields{
			"method":      c.Request.Method,
			"path":        c.Request.URL.Path,
			"status":      c.Writer.Status(),
			"duration_ms": time.Since(start).Milliseconds(),
		}).Info("請求完成")
	}
}
