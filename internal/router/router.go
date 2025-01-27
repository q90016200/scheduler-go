package router

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/q90016200/scheduler-go/internal/app"
	"github.com/q90016200/scheduler-go/internal/controllers"
	"github.com/q90016200/scheduler-go/internal/middleware"
	"time"
)

// SetupRouter 設定所有路由
func SetupRouter(r *gin.Engine, services *app.AppServices) *gin.Engine {
	taskController := controllers.NewTaskController(services)
	authController := controllers.NewAuthController(services)

	api := r.Group("/api")
	taskRoutes := api.Group("/tasks").Use(middleware.OpenTracingMiddleware())
	{
		taskRoutes.GET("/", taskController.ListTasksHandler)
		taskRoutes.POST("/", middleware.JWTAuthMiddleware(), taskController.CreateTaskHandler)
		taskRoutes.PUT("/:id", middleware.JWTAuthMiddleware(), taskController.UpdateTaskHandler)
		taskRoutes.DELETE("/:id", middleware.JWTAuthMiddleware(), taskController.DeleteTaskHandler)
	}

	// 用戶驗證路由
	authRoutes := api.Group("/auth").Use(middleware.OpenTracingMiddleware())
	{
		authRoutes.POST("/register", authController.RegisterHandler)
		authRoutes.POST("/login", authController.LoginHandler)
		authRoutes.POST("/logout", middleware.JWTAuthMiddleware(), authController.LogoutHandler)
	}

	testRoutes := api.Group("/test").Use(middleware.OpenTracingMiddleware())
	{
		testRoutes.Any("/:id", func(c *gin.Context) {
			fmt.Println(c.Param("id"), time.Now())
			c.String(200, "test")
		})
	}

	return r
}
