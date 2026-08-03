package ssrf

import (
	"context"
	"net"
	"testing"

	"github.com/q90016200/scheduler-go/config"
)

func TestIsBlockedSSRFTarget(t *testing.T) {
	blocked := []string{"127.0.0.1", "169.254.169.254", "0.0.0.0", "::1", "fe80::1"}
	for _, ip := range blocked {
		if !isBlockedSSRFTarget(net.ParseIP(ip)) {
			t.Errorf("expected %s to be blocked", ip)
		}
	}

	allowed := []string{"10.0.0.1", "192.168.1.1", "8.8.8.8", "172.16.0.1"}
	for _, ip := range allowed {
		if isBlockedSSRFTarget(net.ParseIP(ip)) {
			t.Errorf("expected %s (private/public, non-loopback) to be allowed", ip)
		}
	}
}

func TestSafeDialContextBlocksLoopback(t *testing.T) {
	config.AppConfig.SSRFAllowPrivateTargets = false
	_, err := SafeDialContext(context.Background(), "tcp", "127.0.0.1:80")
	if err == nil {
		t.Fatal("expected loopback dial to be blocked")
	}
}

func TestSafeDialContextAllowFlagBypasses(t *testing.T) {
	config.AppConfig.SSRFAllowPrivateTargets = true
	defer func() { config.AppConfig.SSRFAllowPrivateTargets = false }()

	// 允許旗標開啟時不應在 Control 階段擋下，錯誤（若有）只會來自實際 connect 失敗
	conn, err := SafeDialContext(context.Background(), "tcp", "127.0.0.1:1")
	if conn != nil {
		conn.Close()
	}
	if err != nil && err.Error() == "blocked SSRF target: 127.0.0.1" {
		t.Fatal("expected SSRFAllowPrivateTargets=true to bypass the block")
	}
}
