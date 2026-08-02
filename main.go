package main

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/q90016200/scheduler-go/config"
	"github.com/q90016200/scheduler-go/internal/app"
	"github.com/q90016200/scheduler-go/internal/database"
	"github.com/q90016200/scheduler-go/internal/router"
	"github.com/q90016200/scheduler-go/internal/scheduler"
	"github.com/q90016200/scheduler-go/pkg/logger"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// 載入配置
	config.LoadConfig(".env")

	// 初始化日誌
	logger.Init("logs/app.log")
	logger.Log.Info("日誌初始化完成")

	// 初始化資料庫
	db, err := database.InitializeDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// 初始化排程器
	cronScheduler := scheduler.NewCronScheduler(db)

	// 初始化 AppServices
	services := app.NewAppServices(db, cronScheduler)

	// 初始化 Gin 引擎 & 註冊路由
	r := gin.Default()
	r = router.SetupRouter(r, services)

	// 啟動排程器
	cronScheduler.StartScheduler()

	// 啟動 HTTP 服務，設定逾時避免慢速連線（Slowloris）長期占用
	port := config.AppConfig.ServerPort
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("服務啟動中，端口：%s", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("服務啟動失敗: %v", err)
		}
	}()

	// 等待信號以安全退出
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Println("服務關閉中...")

	cronScheduler.StopCronScheduler()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("服務關閉失敗: %v", err)
	}
	log.Println("服務已關閉")
}
