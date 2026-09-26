package main

import (
	"fmt"
	"github.com/q90016200/scheduler-go/config"
	"github.com/q90016200/scheduler-go/internal/database"
	"github.com/q90016200/scheduler-go/internal/models"
	"log"
)

func main() {
	// 載入配置
	config.LoadConfig()

	// 初始化資料庫
	db, err := database.InitializeDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// 執行遷移
	fmt.Println("Running database migrations...")
	err = db.AutoMigrate(&models.User{}, &models.Task{}, &models.TaskGroup{}, &models.TaskExecutionLog{})
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	fmt.Println("Database migrations completed.")
}
