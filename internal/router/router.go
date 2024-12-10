package router

import (
	"github.com/gin-gonic/gin"
	"github.com/q90016200/schedule-go/internal/app"
	"github.com/q90016200/schedule-go/internal/controllers"
	"github.com/q90016200/schedule-go/internal/middleware"
)

// SetupRouter 設定所有路由
func SetupRouter(r *gin.Engine, services *app.AppServices) *gin.Engine {
	taskController := controllers.NewTaskController(services)
	authController := controllers.NewAuthController(services)

	r.Group("/api").Use(middleware.OpenTracingMiddleware())
	{
		taskRoutes := r.Group("/tasks")
		{
			taskRoutes.GET("/", taskController.ListTasksHandler)
			taskRoutes.POST("/", middleware.JWTAuthMiddleware(), taskController.CreateTaskHandler)
			taskRoutes.PUT("/:id", middleware.JWTAuthMiddleware(), taskController.UpdateTaskHandler)
			taskRoutes.DELETE("/:id", middleware.JWTAuthMiddleware(), taskController.DeleteTaskHandler)
		}

		// 用戶驗證路由
		authRoutes := r.Group("/auth")
		{
			authRoutes.POST("/register", authController.RegisterHandler)
			authRoutes.POST("/login", authController.LoginHandler)
			authRoutes.POST("/logout", middleware.JWTAuthMiddleware(), authController.LogoutHandler)
		}
	}

	return r
}
