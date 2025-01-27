package logger

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/natefinch/lumberjack"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"path"
)

const (
	red    = 31
	yellow = 33
	blue   = 36
	gray   = 37
)

// Log 全域變數
var Log *logrus.Logger

type LogFormatter struct {
}

func (t LogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	//根据不同的level展示颜色
	var levelColor int
	switch entry.Level {
	case logrus.DebugLevel, logrus.TraceLevel:
		levelColor = gray
	case logrus.WarnLevel:
		levelColor = yellow
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		levelColor = red
	default:
		levelColor = blue
	}
	// 字节缓冲区
	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}
	//自定义日期格式
	timestamp := entry.Time.Format("2006-01-02 15:04:06")
	if entry.HasCaller() {
		//自定义文件路径
		funcVal := entry.Caller.Function
		fileVal := fmt.Sprintf("%s:%d", path.Base(entry.Caller.File), entry.Caller.Line)
		//自定义输出格式
		fmt.Fprintf(b, "[%s] \x1b[%dm[%s]\x1b[0m %s %s %s\n", timestamp, levelColor, entry.Level, fileVal, funcVal, entry.Message)
	} else {
		fmt.Fprintf(b, "[%s] \x1b[%dm[%s]\x1b[0m  %s\n", timestamp, levelColor, entry.Level, entry.Message)
	}
	return b.Bytes(), nil
}

// Init 初始化日誌設置
func Init(logFilePath string) *logrus.Logger {
	Log = logrus.New()

	// 設定輪替檔案
	logFile := &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    10, // 每個檔案最大大小（MB）
		MaxBackups: 5,  // 保留的備份數量
		MaxAge:     10, // 保留的天數
		Compress:   true,
	}
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	Log.SetOutput(multiWriter)

	// 啟用 Caller 報告，開啟回傳函數名稱和行號
	Log.SetReportCaller(true)
	Log.SetFormatter(&LogFormatter{}) //设置自定义的Formatter
	Log.SetLevel(logrus.DebugLevel)   //设置最低等级

	return Log
}

func WithTrace(c *gin.Context) *logrus.Entry {
	traceID, exists := c.Get("trace_id")
	if !exists {
		traceID = "unknown" // 如果沒有 trace_id，則默認為 unknown
	}

	// 返回包含 trace_id 的 Logrus Entry
	return Log.WithField("trace_id", traceID)
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
