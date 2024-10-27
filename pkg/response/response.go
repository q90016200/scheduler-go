package response

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// Response 定義通用的 API 回應結構
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	Error   interface{} `json:"error"`
}

// Success 回傳成功的回應
func Success(c *gin.Context, data interface{}, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: message,
		Data:    data,
		Error:   nil,
	})
}

// Fail 回傳失敗的回應
func Fail(c *gin.Context, code int, errMessage string, errDetail interface{}) {
	c.JSON(code, Response{
		Code:    code,
		Message: errMessage,
		Data:    nil,
		Error:   errDetail,
	})
}
