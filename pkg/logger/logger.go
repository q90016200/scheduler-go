package logger

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/natefinch/lumberjack"
	"github.com/q90016200/scheduler-go/config"
	"github.com/sirupsen/logrus"
	"os"
	"path"
	"sort"
	"strings"
)

const (
	red    = 31
	yellow = 33
	blue   = 36
	gray   = 37
)

// Log 全域變數
var Log *logrus.Logger

// LogFormatter 是給終端機（stdout）用的彩色人類可讀格式；
// 檔案輸出改由 fileHook 以 JSON 格式寫入，見 Init。
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
	//自定义日期格式（15:04:05 才是秒；15:04:06 會誤印成年份後兩位）
	timestamp := entry.Time.Format("2006-01-02 15:04:05")
	fields := formatFields(entry.Data)
	if entry.HasCaller() {
		//自定义文件路径
		funcVal := entry.Caller.Function
		fileVal := fmt.Sprintf("%s:%d", path.Base(entry.Caller.File), entry.Caller.Line)
		//自定义输出格式
		fmt.Fprintf(b, "[%s] \x1b[%dm[%s]\x1b[0m %s %s %s%s\n", timestamp, levelColor, entry.Level, fileVal, funcVal, entry.Message, fields)
	} else {
		fmt.Fprintf(b, "[%s] \x1b[%dm[%s]\x1b[0m  %s%s\n", timestamp, levelColor, entry.Level, entry.Message, fields)
	}
	return b.Bytes(), nil
}

// formatFields 將 entry.Data（trace_id、WithError 附加的 error 等）附加在訊息後面。
// 修正前這些欄位從未被輸出，導致 trace_id 追蹤與錯誤內容從未進過 log。
func formatFields(data logrus.Fields) string {
	if len(data) == 0 {
		return ""
	}
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, " %s=%v", k, data[k])
	}
	return b.String()
}

// fileHook 把 log 以 JSON 格式寫入檔案（可用 jq 依 trace_id 過濾），
// 與 stdout 的彩色人類可讀格式分開，避免 ANSI 跳脫序列混進 log 檔。
type fileHook struct {
	writer    *lumberjack.Logger
	formatter logrus.Formatter
}

func (h *fileHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *fileHook) Fire(entry *logrus.Entry) error {
	line, err := h.formatter.Format(entry)
	if err != nil {
		return err
	}
	_, err = h.writer.Write(line)
	return err
}

// resolveLogLevel 由 LOG_LEVEL 決定，未設定時開發環境預設 debug、其餘預設 info
func resolveLogLevel() logrus.Level {
	if levelStr := config.AppConfig.LogLevel; levelStr != "" {
		if level, err := logrus.ParseLevel(levelStr); err == nil {
			return level
		}
	}
	if config.AppConfig.IsDevelopment() {
		return logrus.DebugLevel
	}
	return logrus.InfoLevel
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

	// stdout：彩色人類可讀格式；檔案：JSON（透過 hook），兩者分流避免 ANSI 碼寫進檔案
	Log.SetOutput(os.Stdout)
	Log.SetFormatter(&LogFormatter{})
	Log.AddHook(&fileHook{writer: logFile, formatter: &logrus.JSONFormatter{}})

	// 啟用 Caller 報告，開啟回傳函數名稱和行號
	Log.SetReportCaller(true)
	Log.SetLevel(resolveLogLevel())

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
