package router

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/q90016200/scheduler-go/internal/app"
	"github.com/q90016200/scheduler-go/internal/controllers"
	"github.com/q90016200/scheduler-go/internal/middleware"
	"github.com/q90016200/scheduler-go/pkg/response"
	"gorm.io/gorm"
	"net/http"
	"strings"
)

// SetupRouter 設定所有路由
func SetupRouter(r *gin.Engine, services *app.AppServices, db *gorm.DB) *gin.Engine {
	taskController := controllers.NewTaskController(services)
	authController := controllers.NewAuthController(services)
	execLogController := controllers.NewTaskExecutionLogController(services)
	userController := controllers.NewUserController(services)
	notifyController := controllers.NewNotifyController()
	tracing := middleware.OpenTracingMiddleware()

	// 健康檢查給容器/LB 探測用：DB 連得上才回 200。放 /api 外、不掛 tracing，避免探測洗版日誌
	r.GET("/healthz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		sqlDB, err := db.DB()
		if err == nil {
			err = sqlDB.PingContext(ctx)
		}
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api")
	api.Use(tracing)
	taskRoutes := api.Group("/tasks")
	{
		taskRoutes.GET("/", taskController.ListTasksHandler)
		taskRoutes.GET("/:id", taskController.GetTaskHandler)
		taskRoutes.POST("/", taskController.CreateTaskHandler)
		taskRoutes.PUT("/:id", taskController.UpdateTaskHandler)
		taskRoutes.DELETE("/:id", taskController.DeleteTaskHandler)
		taskRoutes.PATCH("/:id/status", taskController.UpdateTaskStatusHandler)
		taskRoutes.POST("/:id/run", taskController.RunTaskHandler)
		taskRoutes.DELETE("/:id/run", taskController.CancelTaskRunHandler)
		taskRoutes.GET("/:id/run/:batch_id", taskController.GetRunBatchHandler)

		taskRoutes.GET("/group/", taskController.ListTaskGroupsHandler)
		taskRoutes.GET("/group/:id", taskController.GetTaskGroupHandler)
		taskRoutes.POST("/group", taskController.CreateTaskGroupHandler)
		taskRoutes.PUT("/group/:id", taskController.UpdateTaskGroupHandler)
		taskRoutes.DELETE("/group/:id", taskController.DeleteTaskGroupHandler)

		taskRoutes.GET("/logs/", execLogController.ListLogsHandler)
		taskRoutes.GET("/logs/:id", execLogController.GetLogHandler)
		taskRoutes.GET("/:id/stats", execLogController.GetTaskStatsHandler)
	}

	// 用戶驗證路由
	authRoutes := api.Group("/auth")
	{
		authRoutes.POST("/register", authController.RegisterHandler)
		authRoutes.POST("/login", authController.LoginHandler)
		authRoutes.POST("/logout", authController.LogoutHandler)
		authRoutes.GET("/me", authController.MeHandler)
	}

	dashboardRoutes := api.Group("/dashboard")
	{
		dashboardRoutes.GET("/stats", execLogController.GetDashboardStatsHandler)
	}

	notifyRoutes := api.Group("/notify")
	{
		notifyRoutes.GET("/config", notifyController.GetNotifyConfigHandler)
	}

	// 使用者管理路由
	userRoutes := api.Group("/users")
	{
		userRoutes.GET("/", userController.ListUsersHandler)
		userRoutes.GET("/:id", userController.GetUserHandler)
		userRoutes.POST("/", userController.CreateUserHandler)
		userRoutes.PUT("/:id", userController.UpdateUserHandler)
		userRoutes.PATCH("/:id/status", userController.UpdateUserStatusHandler)
		userRoutes.DELETE("/:id", userController.DeleteUserHandler)
	}

	// 服務前端建置產物（web/dist），同源部署不需要額外 nginx
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
