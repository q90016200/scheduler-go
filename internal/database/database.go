package database

import (
	"fmt"
	"time"

	"github.com/q90016200/scheduler-go/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
)

var DB *gorm.DB

// InitializeDB 初始化資料庫連接
func InitializeDB() (*gorm.DB, error) {
	pg := config.AppConfig.PostgresConfig

	// 資料庫連接字串
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=Asia/Taipei",
		pg.Host,
		pg.Port,
		pg.Username,
		pg.Password,
		pg.Database,
		pg.SSLMode,
	)

	log.Printf("Connecting to database %s:%s/%s", pg.Host, pg.Port, pg.Database)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		return nil, err
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(pg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(pg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(pg.ConnMaxLifetimeMinutes) * time.Minute)

	return DB, nil
}
