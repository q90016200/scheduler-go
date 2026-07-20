package database

import (
	"fmt"
	"github.com/q90016200/scheduler-go/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
)

var DB *gorm.DB

// InitializeDB 初始化資料庫連接
func InitializeDB() (*gorm.DB, error) {
	// 資料庫連接字串
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=Asia/Taipei",
		config.AppConfig.PostgresConfig.Host,
		config.AppConfig.PostgresConfig.Port,
		config.AppConfig.PostgresConfig.Username,
		config.AppConfig.PostgresConfig.Password,
		config.AppConfig.PostgresConfig.Database,
	)

	log.Printf("Connecting to database %s:%s/%s", config.AppConfig.PostgresConfig.Host, config.AppConfig.PostgresConfig.Port, config.AppConfig.PostgresConfig.Database)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
		return nil, err
	}
	return DB, nil
}
