package utils

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/robfig/cron/v3"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var (
	MyValidator *validator.Validate
)

func init() {
	MyValidator = validator.New()
	// 錯誤回報用 json 欄位名（endpoint）而非 Go 欄位名（EndPoint），前端才能直接對應表單欄位
	MyValidator.RegisterTagNameFunc(func(f reflect.StructField) string {
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "" || name == "-" {
			return f.Name
		}
		return name
	})
	// 自定义验证方法
	MyValidator.RegisterValidation("alphanumLength", checkAlphanumLength)
	MyValidator.RegisterValidation("length", checkLength)
	MyValidator.RegisterValidation("prec", checkPrecision)
	MyValidator.RegisterValidation("cron", checkCronExpr)
}

// checkCronExpr 驗證欄位是否為合法的標準 cron 表達式（與 scheduler 實際使用的解析器一致）
func checkCronExpr(fl validator.FieldLevel) bool {
	_, err := cron.ParseStandard(fl.Field().String())
	return err == nil
}

// ValidateTaskMethod 依 Task.Type 檢查 Method 欄位：
// http 任務的 Method 必須是 HTTP 動詞；rpc 任務的 Method 視為 "package.Service/MethodName"，只要求非空。
// 註冊於 internal/models/validation.go，函式留在此處以避免 pkg/utils 依賴 internal/models。
func ValidateTaskMethod(sl validator.StructLevel) {
	current := sl.Current()
	taskType := current.FieldByName("Type").String()
	methodField := current.FieldByName("Method")
	method := methodField.String()

	switch strings.ToLower(taskType) {
	case "http":
		switch method {
		case "GET", "POST", "PUT", "DELETE", "PATCH":
		default:
			sl.ReportError(methodField.Interface(), "method", "Method", "oneof", "GET POST PUT DELETE PATCH")
		}
	case "rpc":
		if method == "" {
			sl.ReportError(methodField.Interface(), "method", "Method", "required", "")
			return
		}
		if _, _, err := SplitRpcMethod(method); err != nil {
			sl.ReportError(methodField.Interface(), "method", "Method", "rpc_method_format", "")
		}
	}
}

// SplitRpcMethod 將 "package.Service/MethodName"（允許帶前導 "/"）拆成 service 全名與方法名稱。
func SplitRpcMethod(method string) (serviceName string, methodName string, err error) {
	method = strings.TrimPrefix(method, "/")
	idx := strings.LastIndex(method, "/")
	if idx <= 0 || idx == len(method)-1 {
		return "", "", fmt.Errorf("method must be in the form \"package.Service/MethodName\", got %q", method)
	}
	return method[:idx], method[idx+1:], nil
}

// FormatValidationErrors 將 validator.ValidationErrors 轉成前端可逐欄位解析的結構化陣列
func FormatValidationErrors(err error) []gin.H {
	var out []gin.H
	if ve, ok := err.(validator.ValidationErrors); ok {
		for _, fe := range ve {
			out = append(out, gin.H{
				"field": fe.Field(),
				"tag":   fe.Tag(),
				"param": fe.Param(),
			})
		}
		return out
	}
	return []gin.H{{"error": err.Error()}}
}

func checkPrecision(fl validator.FieldLevel) bool {
	field := fmt.Sprintf("%.4f", fl.Field().Float())
	field = strings.TrimRight(field, "0")
	param := fl.Param()
	re := fmt.Sprintf("^\\d{1,}\\.?\\d{0,%s}$", param)
	r := regexp.MustCompile(re)
	return r.MatchString(field)
}

func checkAlphanumLength(fl validator.FieldLevel) bool {
	field := fl.Field().String()
	if len(field) == 0 {
		return true
	}
	param := strings.Split(fl.Param(), "/")
	re := fmt.Sprintf("^[a-zA-Z0-9_-]{%s,%s}$", param[0], param[1])
	r := regexp.MustCompile(re)
	return r.MatchString(field)
}

func checkLength(fl validator.FieldLevel) bool {
	field := fl.Field().Int()
	strField := strconv.Itoa(int(field))
	param := fl.Param()
	re := fmt.Sprintf("\\w{%s,%s}", param, param)
	r := regexp.MustCompile(re)
	return r.MatchString(strField)
}
