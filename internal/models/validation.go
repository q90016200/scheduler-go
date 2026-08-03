package models

import "github.com/q90016200/scheduler-go/pkg/utils"

func init() {
	utils.MyValidator.RegisterStructValidation(utils.ValidateTaskMethod, CreateTaskReq{}, UpdateTaskReq{})
}
