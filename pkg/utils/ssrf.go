package utils

import (
	"context"
	"fmt"
	"net"
	"syscall"

	"github.com/q90016200/scheduler-go/config"
)

// SafeDialContext 是給任務執行用的 DialContext，禁止連線到 loopback、link-local
// （含雲端 metadata 位址 169.254.0.0/16）與未指定位址，避免無 auth 的排程服務被當作
// SSRF 跳板探測敏感位址。刻意不封鎖 RFC1918 私有網段——本服務是內部排程工具，
// 打內網服務是其本職功能。
//
// 檢查在 syscall connect 前、對已解析出的實際 IP 進行（而非在建立任務時解析網域），
// 避免僅檢查網域名稱被 DNS rebinding 繞過。
func SafeDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	dialer := &net.Dialer{Control: safeDialControl}
	return dialer.DialContext(ctx, network, addr)
}

func safeDialControl(_, address string, _ syscall.RawConn) error {
	if config.AppConfig.SSRFAllowPrivateTargets {
		return nil
	}
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return fmt.Errorf("invalid IP address: %s", host)
	}
	if isBlockedSSRFTarget(ip) {
		return fmt.Errorf("blocked SSRF target: %s", ip)
	}
	return nil
}

func isBlockedSSRFTarget(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()
}
