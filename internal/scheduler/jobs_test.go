package scheduler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/jhump/protoreflect/desc"
	"github.com/q90016200/scheduler-go/internal/models"
	"github.com/q90016200/scheduler-go/pkg/logger"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func newTestScheduler() *CronScheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &CronScheduler{
		Cron:             cron.New(),
		Log:              &logger.LogrusAdapter{Logger: logrus.New()},
		taskCronEntryIDs: make(map[string]cron.EntryID),
		restyClient:      resty.New(),
		runBatches:       make(map[string]runBatch),
		baseCtx:          ctx,
		baseCancel:       cancel,
		rpcConns:         make(map[string]*grpc.ClientConn),
		rpcMethods:       make(map[string]*desc.MethodDescriptor),
	}
}

func TestApplyLockedEnabledRegisters(t *testing.T) {
	cs := newTestScheduler()
	task := &models.Task{ID: "t1", Type: "http", Schedule: "* * * * *", Method: "GET", Endpoint: "http://example.com", Status: models.STATUS_ENABLED}

	if err := cs.applyLocked(task.ID, task); err != nil {
		t.Fatalf("applyLocked returned error: %v", err)
	}
	if _, exists := cs.taskCronEntryIDs[task.ID]; !exists {
		t.Fatal("expected task to be registered in taskCronEntryIDs")
	}

	// 再 apply 一次：取代舊 entry，不會多出第二個
	if err := cs.applyLocked(task.ID, task); err != nil {
		t.Fatalf("second applyLocked returned error: %v", err)
	}
	if got := len(cs.Cron.Entries()); got != 1 {
		t.Fatalf("expected exactly 1 cron entry after re-applying, got %d", got)
	}
	if got := len(cs.taskCronEntryIDs); got != 1 {
		t.Fatalf("expected exactly 1 map entry after re-applying, got %d", got)
	}
}

func TestApplyLockedRemoves(t *testing.T) {
	cs := newTestScheduler()
	task := &models.Task{ID: "t2", Type: "http", Schedule: "* * * * *", Method: "GET", Endpoint: "http://example.com", Status: models.STATUS_ENABLED}

	// disabled 與 nil（任務已不存在）都要移除既有 entry
	for name, next := range map[string]*models.Task{
		"disabled": {ID: task.ID, Type: task.Type, Schedule: task.Schedule, Status: models.STATUS_DISABLED},
		"nil":      nil,
	} {
		if err := cs.applyLocked(task.ID, task); err != nil {
			t.Fatalf("%s: register: %v", name, err)
		}
		if err := cs.applyLocked(task.ID, next); err != nil {
			t.Fatalf("%s: applyLocked returned error: %v", name, err)
		}
		if _, exists := cs.taskCronEntryIDs[task.ID]; exists {
			t.Fatalf("%s: expected entry removed from taskCronEntryIDs", name)
		}
		if got := len(cs.Cron.Entries()); got != 0 {
			t.Fatalf("%s: expected 0 cron entries, got %d", name, got)
		}
	}
}

func TestApplyLockedUnknownType(t *testing.T) {
	cs := newTestScheduler()
	task := &models.Task{ID: "t3", Type: "shell", Schedule: "* * * * *", Status: models.STATUS_ENABLED}

	if err := cs.applyLocked(task.ID, task); err == nil {
		t.Fatal("expected error for unknown task type")
	}
	if _, exists := cs.taskCronEntryIDs[task.ID]; exists {
		t.Fatal("expected no map entry for an unknown task type")
	}
	if got := len(cs.Cron.Entries()); got != 0 {
		t.Fatalf("expected 0 cron entries, got %d", got)
	}
}

func TestApplyLockedInvalidScheduleDropsOldEntry(t *testing.T) {
	cs := newTestScheduler()
	good := &models.Task{ID: "t2b", Type: "http", Schedule: "* * * * *", Method: "GET", Endpoint: "http://example.com", Status: models.STATUS_ENABLED}
	if err := cs.applyLocked(good.ID, good); err != nil {
		t.Fatal(err)
	}
	bad := *good
	bad.Schedule = "not-a-cron-expr"
	if err := cs.applyLocked(bad.ID, &bad); err == nil {
		t.Fatal("expected error for invalid cron expression")
	}
	if _, exists := cs.taskCronEntryIDs[good.ID]; exists {
		t.Fatal("expected stale entry removed when re-registration fails")
	}
}

func TestRunTaskNowUnknownType(t *testing.T) {
	cs := newTestScheduler()
	task := &models.Task{ID: "t4", Type: "shell"}

	if _, err := RunTaskNow(cs, task, 1, 100); err == nil {
		t.Fatal("expected error for unknown task type")
	}
}

func TestRunTaskNowRepeatsAndRejectsConcurrentBatch(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cs := newTestScheduler()
	task := &models.Task{ID: "batch1", Type: "http", Method: "GET", Endpoint: srv.URL}

	batchID, err := RunTaskNow(cs, task, 3, 20)
	if err != nil {
		t.Fatalf("RunTaskNow returned error: %v", err)
	}
	if batchID == "" || !IsBatchRunning(cs, task.ID, batchID) {
		t.Fatalf("expected batch %q to be running right after RunTaskNow", batchID)
	}
	if IsBatchRunning(cs, task.ID, "other") || IsBatchRunning(cs, "other", batchID) {
		t.Fatal("IsBatchRunning must match both taskID and batchID")
	}
	if _, err := RunTaskNow(cs, task, 3, 20); !errors.Is(err, ErrBatchAlreadyRunning) {
		t.Fatalf("expected ErrBatchAlreadyRunning for concurrent batch, got %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for hits.Load() < 3 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if got := hits.Load(); got != 3 {
		t.Fatalf("expected 3 requests, got %d", got)
	}

	// 批次已跑完，registry 應已清空（goroutine 的 defer 晚於最後一次 runOnce，稍等它收尾）
	for IsBatchRunning(cs, task.ID, batchID) && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if IsBatchRunning(cs, task.ID, batchID) {
		t.Fatal("batch should no longer be running after finishing")
	}
	if err := CancelTaskRun(cs, task.ID); !errors.Is(err, ErrBatchNotFound) {
		t.Fatalf("expected ErrBatchNotFound after batch finished, got %v", err)
	}
}

func TestRunTaskNowRejectsOverGlobalLimit(t *testing.T) {
	cs := newTestScheduler()
	for i := range maxRunBatches {
		cs.runBatches[fmt.Sprint("busy", i)] = runBatch{id: "x", cancel: func() {}}
	}
	task := &models.Task{ID: "over", Type: "http", Method: "GET", Endpoint: "http://example.invalid"}
	if _, err := RunTaskNow(cs, task, 1, 100); !errors.Is(err, ErrTooManyBatches) {
		t.Fatalf("expected ErrTooManyBatches, got %v", err)
	}
	if _, exists := cs.runBatches[task.ID]; exists {
		t.Fatal("rejected batch must not be registered")
	}
}

func TestCancelTaskRunStopsRemainingRequests(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cs := newTestScheduler()
	task := &models.Task{ID: "batch2", Type: "http", Method: "GET", Endpoint: srv.URL}

	if _, err := RunTaskNow(cs, task, 10, 200); err != nil {
		t.Fatalf("RunTaskNow returned error: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for hits.Load() < 1 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if err := CancelTaskRun(cs, task.ID); err != nil {
		t.Fatalf("CancelTaskRun returned error: %v", err)
	}

	time.Sleep(400 * time.Millisecond) // 遠超第二次 interval，若取消沒生效這裡就會抓到第二筆
	if got := hits.Load(); got != 1 {
		t.Fatalf("expected exactly 1 request after cancel, got %d", got)
	}
}

func TestDoHTTPAttempt4xxNotRetried(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	cs := newTestScheduler()
	task := &models.Task{ID: "http4xx", Type: "http", Method: "GET", Endpoint: srv.URL, RetryCount: 2, RetryDelayMs: 0}

	runOnce, err := cs.runnerFor(task, models.SOURCE_CRON, "")
	if err != nil {
		t.Fatalf("runnerFor returned error: %v", err)
	}
	runOnce(context.Background())

	if got := hits.Load(); got != 1 {
		t.Fatalf("expected exactly 1 request for a non-retryable 4xx, got %d", got)
	}
}

func TestDoHTTPAttempt5xxRetried(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	cs := newTestScheduler()
	task := &models.Task{ID: "http5xx", Type: "http", Method: "GET", Endpoint: srv.URL, RetryCount: 2, RetryDelayMs: 0}

	runOnce, err := cs.runnerFor(task, models.SOURCE_CRON, "")
	if err != nil {
		t.Fatalf("runnerFor returned error: %v", err)
	}
	runOnce(context.Background())

	if got := hits.Load(); got != 3 {
		t.Fatalf("expected 3 requests (1 + 2 retries) for a retryable 5xx, got %d", got)
	}
}

func TestDoHTTPAttemptSuccessStatusCodes(t *testing.T) {
	cases := []struct {
		spec          string
		code          int
		wantStatus    string
		wantRetryable bool
	}{
		{"", 202, models.EXEC_STATUS_SUCCESS, false},
		{"200", 202, models.EXEC_STATUS_FAILED, false},
		{"200,2xx", 202, models.EXEC_STATUS_SUCCESS, false},
		{"404", 404, models.EXEC_STATUS_SUCCESS, false},
		{"200", 503, models.EXEC_STATUS_FAILED, true},
	}
	for _, c := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(c.code)
		}))
		task := &models.Task{Method: "GET", Endpoint: srv.URL, SuccessStatusCodes: c.spec}
		status, _, _, retryable := newTestScheduler().doHTTPAttempt(context.Background(), task, nil)
		srv.Close()
		if status != c.wantStatus || retryable != c.wantRetryable {
			t.Errorf("spec=%q code=%d: got (%s, %v), want (%s, %v)", c.spec, c.code, status, retryable, c.wantStatus, c.wantRetryable)
		}
	}
}

// 取消 parentCtx 後不得再重試：RetryDelayMs 為 0 或 >0 都只呼叫 attemptFn 一次
func TestRunWithRetryCancelStopsRetries(t *testing.T) {
	for _, delayMs := range []int{0, 50} {
		cs := newTestScheduler() // db 為 nil：logExecution／連續失敗計數皆 no-op
		task := &models.Task{ID: "t", RetryCount: 3, RetryDelayMs: delayMs, TimeoutSeconds: 5}
		ctx, cancel := context.WithCancel(context.Background())

		calls := 0
		cs.runWithRetry(ctx, task, models.SOURCE_CRON, "", func(context.Context) (string, string, string, bool) {
			calls++
			cancel() // 第一次嘗試中被取消；回傳可重試的失敗
			return models.EXEC_STATUS_FAILED, "", "context canceled", true
		})
		if calls != 1 {
			t.Fatalf("RetryDelayMs=%d: attemptFn called %d times, want 1", delayMs, calls)
		}
	}
}

// NextRunAts 只回傳已註冊任務的下次執行時間，未註冊的 id 不在結果內
func TestNextRunAts(t *testing.T) {
	cs := newTestScheduler()
	for _, id := range []string{"a", "b"} {
		task := &models.Task{ID: id, Type: "http", Schedule: "* * * * *", Method: "GET", Endpoint: "http://example.com", Status: models.STATUS_ENABLED}
		if err := cs.applyLocked(id, task); err != nil {
			t.Fatalf("applyLocked %s: %v", id, err)
		}
	}
	// 未 Start 時 entry 的 Next 是零值；Start 後才會算出下次時間
	cs.Cron.Start()
	defer cs.Cron.Stop()
	got := NextRunAts(cs, []string{"a", "b", "missing"})
	if len(got) != 2 {
		t.Fatalf("got %d keys, want 2: %v", len(got), got)
	}
	if _, ok := got["missing"]; ok {
		t.Fatal("unregistered id must not be in result")
	}
	if got["a"].IsZero() || got["b"].IsZero() {
		t.Fatal("registered ids should have non-zero next time")
	}
}
