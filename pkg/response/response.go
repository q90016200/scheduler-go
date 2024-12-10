package response

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// Response 定義通用的 API 回應結構
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   interface{} `json:"error,omitempty"`
	Trace   any         `json:"trace"`
}

// Success 回傳成功的回應
func Success(c *gin.Context, data interface{}, message string) {
	// 取得 Trace ID
	traceID, exists := c.Get("trace_id")
	if !exists {
		traceID = "unknown"
	}

	c.AbortWithStatusJSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: message,
		Data:    data,
		Error:   nil,
		Trace:   traceID,
	})

	return
}

// Fail 回傳失敗的回應
func Fail(c *gin.Context, code int, errMessage string, errDetail interface{}) {
	// 取得 Trace ID
	traceID, exists := c.Get("trace_id")
	if !exists {
		traceID = "unknown"
	}

	c.AbortWithStatusJSON(code, Response{
		Code:    code,
		Message: errMessage,
		Data:    nil,
		Error:   errDetail,
		Trace:   traceID,
	})

	return
}
