package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/q90016200/schedule-go/internal/auth"
	"github.com/q90016200/schedule-go/pkg/response"
	"net/http"
)

func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		//// 從 Authorization 頭中獲取 JWT 令牌
		//tokenString := c.GetHeader("Authorization")
		//if tokenString == "" {
		//	c.JSON(http.StatusUnauthorized, gin.H{"error": "缺少認證標頭"})
		//	c.Abort()
		//	return
		//}

		// 從 Cookie 獲取 JWT
		tokenString, err := c.Cookie("auth_token")
		if err != nil {
			response.Fail(c, http.StatusUnauthorized, "No auth token found", nil)
			return
		}

		claims, err := auth.ParseToken(tokenString)
		if err != nil {
			response.Fail(c, http.StatusUnauthorized, "無效或過期的 Token", nil)
			c.Abort()
			return
		}

		// 將解析出的 UserID 存入請求上下文中
		c.Set("userID", claims.UserID)
		c.Next()
	}
}
