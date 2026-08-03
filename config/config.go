package config

import (
	"github.com/spf13/viper"
	"log"
	"time"
)

// Config 定義應用程式配置
type Config struct {
	AppEnv                    string
	ServerPort                string
	ServerDomain              string
	AppSecret                 string
	PostgresConfig            PostgresConfig
	TokenExpiry               time.Duration
	ExecutionLogRetentionDays int
	SSRFAllowPrivateTargets   bool
}

// PostgresConfig PostgreSQL 配置
type PostgresConfig struct {
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

	appEnv := viper.GetString("APP_ENV")
	if appEnv == "" {
		appEnv = "development"
	}

	executionLogRetentionDays := viper.GetInt("EXECUTION_LOG_RETENTION_DAYS")
	if executionLogRetentionDays <= 0 {
		executionLogRetentionDays = 120
	}

	AppConfig = Config{
		AppEnv:                    appEnv,
		ServerPort:                viper.GetString("SERVER_PORT"),
		ServerDomain:              viper.GetString("SERVER_DOMAIN"),
		AppSecret:                 viper.GetString("APP_SECRET"),
		TokenExpiry:               viper.GetDuration("TOKEN_EXPIRY"),
		ExecutionLogRetentionDays: executionLogRetentionDays,
		SSRFAllowPrivateTargets:   viper.GetBool("SSRF_ALLOW_PRIVATE_TARGETS"),
		PostgresConfig: PostgresConfig{
			Username: viper.GetString("POSTGRES_USER"),
			Password: viper.GetString("POSTGRES_PASSWORD"),
			Host:     viper.GetString("POSTGRES_HOST"),
			Port:     viper.GetString("POSTGRES_PORT"),
			Database: viper.GetString("POSTGRES_DATABASE"),
		},
	}

	requireNonEmpty := map[string]string{
		"APP_SECRET":        AppConfig.AppSecret,
		"POSTGRES_HOST":     AppConfig.PostgresConfig.Host,
		"POSTGRES_PORT":     AppConfig.PostgresConfig.Port,
		"POSTGRES_USER":     AppConfig.PostgresConfig.Username,
		"POSTGRES_PASSWORD": AppConfig.PostgresConfig.Password,
		"POSTGRES_DATABASE": AppConfig.PostgresConfig.Database,
	}
	for key, val := range requireNonEmpty {
		if val == "" {
			log.Fatalf("缺少必要的環境變數: %s", key)
		}
	}
}
