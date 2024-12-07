package config

import (
	"github.com/spf13/viper"
	"log"
	"time"
)

// Config 定義應用程式配置
type Config struct {
	ServerPort   string
	ServerDomain string
	AppSecret    string
	TaskEnabled  bool
	MySqlConfig  MysqlConfig
	TokenExpiry  time.Duration
}

// MysqlConfig MySql 配置
type MysqlConfig struct {
	Username string
	Password string
	Host     string
	Port     string
	Database string
}

// AppConfig 全域配置變數
var AppConfig Config

// LoadConfig 載入配置
func LoadConfig(filePath string) {
	viper.SetConfigFile(filePath) // 配置文件名稱
	viper.AutomaticEnv()          // 支援環境變數

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("沒有找到 .env 配置文件，將使用環境變數: %v", err)
	}

	AppConfig = Config{
		ServerPort:   viper.GetString("SERVER_PORT"),
		ServerDomain: viper.GetString("SERVER_DOMAIN"),
		TaskEnabled:  viper.GetBool("TASK_ENABLED"),
		AppSecret:    viper.GetString("APP_SECRET"),
		TokenExpiry:  viper.GetDuration("TOKEN_EXPIRY"),
		MySqlConfig: MysqlConfig{
			Username: viper.GetString("MYSQL_USER"),
			Password: viper.GetString("MYSQL_PASSWORD"),
			Host:     viper.GetString("MYSQL_HOST"),
			Port:     viper.GetString("MYSQL_PORT"),
			Database: viper.GetString("MYSQL_DATABASE"),
		},
	}
}
