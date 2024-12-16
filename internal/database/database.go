package database

import (
	"github.com/q90016200/scheduler-go/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"log"
)

var DB *gorm.DB

// InitializeDB 初始化資料庫連接
func InitializeDB() (*gorm.DB, error) {
	// 資料庫連接字串
	dsn := config.AppConfig.MySqlConfig.Username + ":" + config.AppConfig.MySqlConfig.Password + "@tcp(" + config.AppConfig.MySqlConfig.Host + ":" + config.AppConfig.MySqlConfig.Port + ")/" + config.AppConfig.MySqlConfig.Database + "?charset=utf8mb4&parseTime=true&loc=Local"

	log.Printf("Connecting to database %s", dsn)

	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
		return nil, err
	}
	return DB, nil
}
