package router

import (
	"github.com/gin-gonic/gin"
	"github.com/q90016200/schedule-go/internal/app"
	"github.com/q90016200/schedule-go/internal/controllers"
)

// SetupRouter 設定所有路由
func SetupRouter(r *gin.Engine, services *app.AppServices) *gin.Engine {
	taskController := controllers.NewTaskController(services)

	// 用戶驗證路由
	//authRoutes := r.Group("/auth")
	//{
	//	authRoutes.POST("/login", auth.Login)
	//	authRoutes.POST("/register", auth.Register)
	//}

	taskRoutes := r.Group("/tasks")
	{
		taskRoutes.GET("/", taskController.ListTasksHandler)
		taskRoutes.POST("/", taskController.CreateTaskHandler)
		taskRoutes.PUT("/:id", taskController.UpdateTaskHandler)
		taskRoutes.DELETE("/:id", taskController.DeleteTaskHandler)
	}

	return r
}
