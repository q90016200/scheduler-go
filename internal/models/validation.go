package models

import "github.com/q90016200/scheduler-go/pkg/utils"

func init() {
	// UpdateTaskReq 是 CreateTaskReq 的型別別名（同一個型別），只需註冊一次
	utils.MyValidator.RegisterStructValidation(utils.ValidateTaskMethod, CreateTaskReq{})
}
