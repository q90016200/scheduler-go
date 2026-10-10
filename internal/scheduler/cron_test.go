package scheduler

import (
	"testing"

	"google.golang.org/grpc/connectivity"
)

// 回歸：resty client 不得設定 http.Client.Timeout，否則會在 task.TimeoutSeconds 之前截斷長任務
func TestSSRFGuardedRestyClientHasNoClientTimeout(t *testing.T) {
	client := newSSRFGuardedRestyClient()
	if got := client.GetClient().Timeout; got != 0 {
		t.Fatalf("http.Client.Timeout = %v, want 0 (timeout must come from per-attempt context)", got)
	}
}

// gRPC 連線快取：同 endpoint 重複取得同一條 conn；Stop 後 conn 關閉且不再建立新連線
func TestRPCConnCacheAndClose(t *testing.T) {
	cs := newTestScheduler()

	c1, err := cs.rpcConn("127.0.0.1:1")
	if err != nil {
		t.Fatalf("rpcConn: %v", err)
	}
	c2, err := cs.rpcConn("127.0.0.1:1")
	if err != nil {
		t.Fatalf("rpcConn: %v", err)
	}
	if c1 != c2 {
		t.Fatal("same endpoint should return the cached conn")
	}
	if other, _ := cs.rpcConn("127.0.0.1:2"); other == c1 {
		t.Fatal("different endpoint should get its own conn")
	}

	cs.StopCronScheduler()

	if got := c1.GetState(); got != connectivity.Shutdown {
		t.Fatalf("conn state after Stop = %v, want Shutdown", got)
	}
	if _, err := cs.rpcConn("127.0.0.1:1"); err == nil {
		t.Fatal("rpcConn after Stop should return an error")
	}
}
