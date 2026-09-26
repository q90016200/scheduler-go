package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/q90016200/scheduler-go/internal/auth"
	"github.com/q90016200/scheduler-go/pkg/response"
	"net/http"
)

// JWTAuthMiddleware 目前未掛載於 router（見 internal/router/router.go），全站 API 暫時開放。
// 啟用時需同時恢復 web/src/router/index.ts 的登入守衛與 App.vue 的登入殼層。
func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 從 Cookie 獲取 JWT
		tokenString, err := c.Cookie("auth_token")
		if err != nil {
			response.Fail(c, http.StatusUnauthorized, "No auth token found", nil)
			return
		}

		claims, err := auth.ParseToken(tokenString)
		if err != nil {
			response.Fail(c, http.StatusUnauthorized, "無效或過期的 Token", nil)
			return
		}

		// 將解析出的 UserID 存入請求上下文中
		c.Set("userID", claims.UserID)
		c.Next()
	}
}
