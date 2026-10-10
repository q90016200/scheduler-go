package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// 回歸：gin 遇到路由衝突會在註冊時 panic，新增 /:id/run/:batch_id 後要確認整組路由仍能註冊，
// 且該路徑確實被路由到 handler（而不是 NoRoute）。
// handler 依賴 services，這裡傳 nil 會在 handler 內 panic；用 gin.Recovery 接住，
// 只要不是 NoRoute 的 404「not found」就代表路由命中。
func TestSetupRouterRoutesRunBatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	SetupRouter(r, nil, nil)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/tasks/x/run/y", nil))

	if w.Code == http.StatusNotFound && strings.Contains(w.Body.String(), "not found") {
		t.Fatalf("GET /api/tasks/x/run/y fell through to NoRoute: %s", w.Body.String())
	}
}
