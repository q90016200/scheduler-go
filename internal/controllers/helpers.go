package controllers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/q90016200/scheduler-go/pkg/logger"
	"github.com/q90016200/scheduler-go/pkg/response"
	"github.com/q90016200/scheduler-go/pkg/utils"
	"gorm.io/gorm"
)

// failErr 統一「err 為 gorm.ErrRecordNotFound → 404，其餘錯誤記 log 後回 500」；
// 404 是呼叫端的問題，不記 log，只有非預期的 500 才記（見 CLAUDE.md Response envelope）
func failErr(c *gin.Context, err error, notFoundMsg, internalMsg string) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.Fail(c, http.StatusNotFound, notFoundMsg, nil)
		return
	}
	logger.WithTrace(c).Error(err.Error())
	response.Fail(c, http.StatusInternalServerError, internalMsg, nil)
}

// bindJSON 綁定並驗證 JSON request body，成功回傳 true；失敗時記 Info log 並寫好 400 回應，呼叫端應立即 return
func bindJSON(c *gin.Context, req any) bool {
	err := c.ShouldBindJSON(req)
	if err == nil {
		err = utils.MyValidator.Struct(req)
	}
	if err == nil {
		return true
	}
	logger.WithTrace(c).Info(err.Error())
	response.Fail(c, http.StatusBadRequest, "參數驗證失敗", utils.FormatValidationErrors(err))
	return false
}

// pageParams 解析 limit/offset 分頁參數：limit 預設 20，超出 1-100 範圍時退回預設值；offset 預設 0，負值退回 0
func pageParams(c *gin.Context) (limit, offset int) {
	limit, _ = strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset, _ = strconv.Atoi(c.DefaultQuery("offset", "0"))
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}
