package main

import (
	"github.com/gin-gonic/gin"
	"github.com/q90016200/schedule-go/config"
	"github.com/q90016200/schedule-go/internal/app"
	"github.com/q90016200/schedule-go/internal/router"
	"github.com/q90016200/schedule-go/pkg/logger"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"log"
)

func main() {
	// 載入配置
	config.LoadConfig()

	// 初始化日誌
	logger.Init()
	logger.Logger.Info("日誌初始化完成")

	// 初始化資料庫
	dsn := config.AppConfig.MySqlConfig.Username + ":" + config.AppConfig.MySqlConfig.Password + "@tcp(" + config.AppConfig.MySqlConfig.Host + ":" + config.AppConfig.MySqlConfig.Port + ")/" + config.AppConfig.MySqlConfig.Database + "?charset=utf8mb4&parseTime=true&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		//logger.Logger.Fatalf("資料庫初始化失敗: %v", err)
	}

	// 初始化 AppServices
	services := app.NewAppServices(db)

	// 初始化 Gin 引擎 & 註冊路由
	r := gin.Default()
	r = router.SetupRouter(r, services)

	// 啟動 HTTP 服務
	port := config.AppConfig.ServerPort
	log.Printf("服務啟動中，端口：%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("服務啟動失敗: %v", err)
	}
}
