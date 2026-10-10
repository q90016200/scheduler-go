package notify

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

// 連線失敗時回傳的錯誤不可含請求 URL（內含 bot token / webhook 密鑰）
func TestDoErrorOmitsURL(t *testing.T) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "http://127.0.0.1:1/botSECRET123/x", nil)
	if err != nil {
		t.Fatal(err)
	}
	err = do(req, "telegram API")
	if err == nil {
		t.Fatal("預期連線失敗")
	}
	if strings.Contains(err.Error(), "SECRET123") {
		t.Fatalf("錯誤訊息洩漏密鑰: %v", err)
	}
}
