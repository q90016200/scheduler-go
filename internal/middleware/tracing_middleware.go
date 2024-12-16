package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/q90016200/scheduler-go/pkg/logger"
	"github.com/q90016200/scheduler-go/pkg/utils"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
)

func OpenTracingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 生成唯一的 Trace ID
		traceID := utils.RandString(3) + xid.New().String()

		// 將 Trace ID 加入到 Context 中
		c.Set("trace_id", traceID)

		// 記錄請求開始的日誌
		log := logger.WithTrace(c)
		log.WithFields(logrus.Fields{
			"method": c.Request.Method,
			"path":   c.Request.URL.Path,
		}).Info("收到新的請求")

		// 處理請求
		c.Next()

		// 記錄請求結束的日誌
		log.Printf("End request, Trace ID: %s", traceID)
	}
}
