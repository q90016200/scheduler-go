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

// write 組出通用回應信封並中止請求；Success/Fail 都只是它的固定參數版本
func write(c *gin.Context, code int, message string, data, errDetail interface{}) {
	// 取得 Trace ID
	traceID, exists := c.Get("trace_id")
	if !exists {
		traceID = "unknown"
	}

	c.AbortWithStatusJSON(code, Response{
		Code:    code,
		Message: message,
		Data:    data,
		Error:   errDetail,
		Trace:   traceID,
	})
}

// Success 回傳成功的回應
func Success(c *gin.Context, data interface{}, message string) {
	write(c, http.StatusOK, message, data, nil)
}

// Fail 回傳失敗的回應
func Fail(c *gin.Context, code int, errMessage string, errDetail interface{}) {
	write(c, code, errMessage, nil, errDetail)
}
