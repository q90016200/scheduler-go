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
	config.LoadConfig("../../.env")

	// 初始化資料庫
	_, err := database.InitializeDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// 執行遷移
	fmt.Println("Running database migrations...")
	err = database.DB.AutoMigrate(&models.User{}, &models.Task{}, &models.TaskGroup{})
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	fmt.Println("Database migrations completed.")
}
