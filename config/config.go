package config

import (
	"log"
	"strings"

	"github.com/spf13/viper"
)

// Config 定義應用程式配置
type Config struct {
	AppEnv                    string
	ServerPort                string
	ServerDomain              string
	AppSecret                 string
	PostgresConfig            PostgresConfig
	TokenExpirySeconds        int
	ExecutionLogRetentionDays int
	SSRFAllowPrivateTargets   bool
	LogLevel                  string
	NotifyConfig              NotifyConfig
}

// NotifyConfig 失敗通知的三個管道設定；任一管道必要欄位為空就視為未啟用
type NotifyConfig struct {
	TelegramBotToken string
	TelegramChatID   string
	SlackWebhookURL  string
	SMTPHost         string
	SMTPPort         string
	SMTPUser         string
	SMTPPassword     string
	SMTPFrom         string
	SMTPTo           string
}

// PostgresConfig PostgreSQL 配置
type PostgresConfig struct {
	Username               string
	Password               string
	Host                   string
	Port                   string
	Database               string
	SSLMode                string
	MaxOpenConns           int
	MaxIdleConns           int
	ConnMaxLifetimeMinutes int
}

// AppConfig 全域配置變數
var AppConfig Config

// IsDevelopment 是否為開發環境
func (c Config) IsDevelopment() bool {
	return c.AppEnv == "development"
}

// LoadConfig 載入配置
func LoadConfig() {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv() // 支援環境變數

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("沒有找到 .env 配置文件，將使用環境變數: %v", err)
	}

	viper.SetDefault("APP_ENV", "development")
	viper.SetDefault("TOKEN_EXPIRY", 604800)
	viper.SetDefault("EXECUTION_LOG_RETENTION_DAYS", 120)
	viper.SetDefault("POSTGRES_SSLMODE", "disable")
	viper.SetDefault("POSTGRES_MAX_OPEN_CONNS", 25)
	viper.SetDefault("POSTGRES_MAX_IDLE_CONNS", 5)
	viper.SetDefault("POSTGRES_CONN_MAX_LIFETIME_MINUTES", 60)

	tokenExpirySeconds := viper.GetInt("TOKEN_EXPIRY")
	if tokenExpirySeconds <= 0 {
		tokenExpirySeconds = 604800
	}

	executionLogRetentionDays := viper.GetInt("EXECUTION_LOG_RETENTION_DAYS")
	if executionLogRetentionDays <= 0 {
		executionLogRetentionDays = 120
	}

	AppConfig = Config{
		AppEnv:                    viper.GetString("APP_ENV"),
		ServerPort:                viper.GetString("SERVER_PORT"),
		ServerDomain:              viper.GetString("SERVER_DOMAIN"),
		AppSecret:                 viper.GetString("APP_SECRET"),
		TokenExpirySeconds:        tokenExpirySeconds,
		ExecutionLogRetentionDays: executionLogRetentionDays,
		SSRFAllowPrivateTargets:   viper.GetBool("SSRF_ALLOW_PRIVATE_TARGETS"),
		LogLevel:                  viper.GetString("LOG_LEVEL"),
		NotifyConfig: NotifyConfig{
			TelegramBotToken: viper.GetString("TELEGRAM_BOT_TOKEN"),
			TelegramChatID:   viper.GetString("TELEGRAM_CHAT_ID"),
			SlackWebhookURL:  viper.GetString("SLACK_WEBHOOK_URL"),
			SMTPHost:         viper.GetString("SMTP_HOST"),
			SMTPPort:         viper.GetString("SMTP_PORT"),
			SMTPUser:         viper.GetString("SMTP_USER"),
			SMTPPassword:     viper.GetString("SMTP_PASSWORD"),
			SMTPFrom:         viper.GetString("SMTP_FROM"),
			SMTPTo:           viper.GetString("SMTP_TO"),
		},
		PostgresConfig: PostgresConfig{
			Username:               viper.GetString("POSTGRES_USER"),
			Password:               viper.GetString("POSTGRES_PASSWORD"),
			Host:                   viper.GetString("POSTGRES_HOST"),
			Port:                   viper.GetString("POSTGRES_PORT"),
			Database:               viper.GetString("POSTGRES_DATABASE"),
			SSLMode:                viper.GetString("POSTGRES_SSLMODE"),
			MaxOpenConns:           viper.GetInt("POSTGRES_MAX_OPEN_CONNS"),
			MaxIdleConns:           viper.GetInt("POSTGRES_MAX_IDLE_CONNS"),
			ConnMaxLifetimeMinutes: viper.GetInt("POSTGRES_CONN_MAX_LIFETIME_MINUTES"),
		},
	}

	required := []struct{ key, val string }{
		{"SERVER_PORT", AppConfig.ServerPort},
		{"APP_SECRET", AppConfig.AppSecret},
		{"POSTGRES_HOST", AppConfig.PostgresConfig.Host},
		{"POSTGRES_PORT", AppConfig.PostgresConfig.Port},
		{"POSTGRES_USER", AppConfig.PostgresConfig.Username},
		{"POSTGRES_PASSWORD", AppConfig.PostgresConfig.Password},
		{"POSTGRES_DATABASE", AppConfig.PostgresConfig.Database},
	}
	var missing []string
	for _, r := range required {
		if r.val == "" {
			missing = append(missing, r.key)
		}
	}
	if len(missing) > 0 {
		log.Fatalf("缺少必要的環境變數: %s", strings.Join(missing, ", "))
	}
}
