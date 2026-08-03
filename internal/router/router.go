package router

import (
	"github.com/gin-gonic/gin"
	"github.com/q90016200/scheduler-go/config"
	"github.com/q90016200/scheduler-go/internal/app"
	"github.com/q90016200/scheduler-go/internal/controllers"
	"github.com/q90016200/scheduler-go/internal/middleware"
	"github.com/q90016200/scheduler-go/pkg/response"
	"net/http"
	"strings"
)

// SetupRouter 設定所有路由
func SetupRouter(r *gin.Engine, services *app.AppServices) *gin.Engine {
	taskController := controllers.NewTaskController(services)
	authController := controllers.NewAuthController(services)
	execLogController := controllers.NewTaskExecutionLogController(services)
	userController := controllers.NewUserController(services)

	// 開發模式下前端走 Vite dev server（不同 origin），需要 CORS；正式環境同源部署不需要
	if config.AppConfig.AppEnv == "development" && len(config.AppConfig.CORSAllowOrigins) > 0 {
		r.Use(middleware.CORSMiddleware(config.AppConfig.CORSAllowOrigins))
	}

	api := r.Group("/api")
	taskRoutes := api.Group("/tasks").Use(middleware.OpenTracingMiddleware())
	{
		taskRoutes.GET("/", taskController.ListTasksHandler)
		taskRoutes.GET("/:id", taskController.GetTaskHandler)
		taskRoutes.POST("/", taskController.CreateTaskHandler)
		taskRoutes.PUT("/:id", taskController.UpdateTaskHandler)
		taskRoutes.DELETE("/:id", taskController.DeleteTaskHandler)
		taskRoutes.PATCH("/:id/status", taskController.UpdateTaskStatusHandler)
		taskRoutes.POST("/:id/run", taskController.RunTaskHandler)

		taskRoutes.GET("/group/", taskController.ListTaskGroupsHandler)
		taskRoutes.GET("/group/:id", taskController.GetTaskGroupHandler)
		taskRoutes.POST("/group", taskController.CreateTaskGroupHandler)
		taskRoutes.PUT("/group/:id", taskController.UpdateTaskGroupHandler)
		taskRoutes.DELETE("/group/:id", taskController.DeleteTaskGroupHandler)

		taskRoutes.GET("/logs/", execLogController.ListLogsHandler)
		taskRoutes.GET("/logs/:id", execLogController.GetLogHandler)
	}

	// 用戶驗證路由
	authRoutes := api.Group("/auth").Use(middleware.OpenTracingMiddleware())
	{
		authRoutes.POST("/register", authController.RegisterHandler)
		authRoutes.POST("/login", authController.LoginHandler)
		authRoutes.POST("/logout", authController.LogoutHandler)
		authRoutes.GET("/me", authController.MeHandler)
	}

	// 使用者管理路由
	userRoutes := api.Group("/users").Use(middleware.OpenTracingMiddleware())
	{
		userRoutes.GET("/", userController.ListUsersHandler)
		userRoutes.GET("/:id", userController.GetUserHandler)
		userRoutes.POST("/", userController.CreateUserHandler)
		userRoutes.PUT("/:id", userController.UpdateUserHandler)
		userRoutes.PATCH("/:id/status", userController.UpdateUserStatusHandler)
		userRoutes.DELETE("/:id", userController.DeleteUserHandler)
	}

	// 服務前端建置產物（web/dist），同源部署不需要 CORS/nginx
	r.Static("/assets", "./web/dist/assets")
	r.StaticFile("/favicon.ico", "./web/dist/favicon.ico")
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			response.Fail(c, http.StatusNotFound, "not found", nil)
			return
		}
		c.File("./web/dist/index.html")
	})

	return r
}
