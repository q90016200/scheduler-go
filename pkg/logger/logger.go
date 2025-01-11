package logger

import (
	"github.com/gin-gonic/gin"
	"github.com/natefinch/lumberjack"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"sync"
	"time"
)

// Logger 全域變數
var Logger *logrus.Logger
var once sync.Once

// 定義自訂 Formatter，將時間轉換為 UTC+8
type utc8Formatter struct {
	Formatter logrus.Formatter
}

func (f *utc8Formatter) Format(entry *logrus.Entry) ([]byte, error) {
	entry.Time = entry.Time.In(time.FixedZone("UTC+8", 8*60*60)) // 設定為 UTC+8
	return f.Formatter.Format(entry)
}

// Init 初始化日誌設置
func Init(logFilePath string) {
	once.Do(func() {
		Logger = logrus.New()

		// 設定輪替檔案
		logFile := &lumberjack.Logger{
			Filename:   logFilePath,
			MaxSize:    10, // 每個檔案最大大小（MB）
			MaxBackups: 5,  // 保留的備份數量
			MaxAge:     10, // 保留的天數
			Compress:   true,
		}

		multiWriter := io.MultiWriter(os.Stdout, logFile)
		Logger.SetOutput(multiWriter)
		// 使用自訂的 Formatter
		Logger.SetFormatter(&utc8Formatter{
			Formatter: &logrus.JSONFormatter{
				TimestampFormat: "2006-01-02T15:04:05", // 設定時間格式
			},
		})
		// 設定日誌等級
		Logger.SetLevel(logrus.InfoLevel)
	})
}

func WithTrace(c *gin.Context) *logrus.Entry {
	traceID, exists := c.Get("trace_id")
	if !exists {
		traceID = "unknown" // 如果沒有 trace_id，則默認為 unknown
	}

	// 返回包含 trace_id 的 Logrus Entry
	return Logger.WithField("trace_id", traceID)
}

// LogrusAdapter 是一個適配 logrus 的 cron.Logger 實現
type LogrusAdapter struct {
	Logger *logrus.Logger
}

// Info 實現 Info 日志
func (l *LogrusAdapter) Info(msg string, keysAndValues ...interface{}) {
	l.Logger.WithFields(convertFields(keysAndValues...)).Info(msg)
}

// Error 實現 Error 日志
func (l *LogrusAdapter) Error(err error, msg string, keysAndValues ...interface{}) {
	l.Logger.WithFields(convertFields(keysAndValues...)).WithError(err).Error(msg)
}

// convertFields 將鍵值對轉換為 logrus.Fields
func convertFields(keysAndValues ...interface{}) logrus.Fields {
	fields := logrus.Fields{}
	for i := 0; i < len(keysAndValues)-1; i += 2 {
		if key, ok := keysAndValues[i].(string); ok {
			fields[key] = keysAndValues[i+1]
		}
	}
	return fields
}
