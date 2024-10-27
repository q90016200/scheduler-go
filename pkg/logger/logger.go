package logger

import (
	"github.com/sirupsen/logrus"
	"os"
)

// Logger 全域變數
var Logger *logrus.Logger

// Init 初始化日誌設置
func Init() {
	Logger = logrus.New()

	// 設定輸出至標準輸出
	Logger.Out = os.Stdout

	// 設定日誌格式
	Logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// 設定日誌等級
	Logger.SetLevel(logrus.InfoLevel)
}

// SetLogLevel 動態設置日誌等級
func SetLogLevel(level logrus.Level) {
	Logger.SetLevel(level)
}
