package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/q90016200/schedule-go/pkg/utils"
	"github.com/rs/xid"
	"log"
)

func OpenTracingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 生成唯一的 Trace ID
		traceID := utils.RandCreator(3) + "-" + xid.New().String()

		// 將 Trace ID 加入到 Context 中
		c.Set("TraceID", traceID)

		// 記錄請求開始的日誌
		log.Printf("Start request, Trace ID: %s", traceID)

		// 處理請求
		c.Next()

		// 記錄請求結束的日誌
		log.Printf("End request, Trace ID: %s", traceID)
	}
}
