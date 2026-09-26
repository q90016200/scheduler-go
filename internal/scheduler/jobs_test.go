package scheduler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/q90016200/scheduler-go/internal/models"
	"github.com/q90016200/scheduler-go/pkg/logger"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

func newTestScheduler() *CronScheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &CronScheduler{
		Cron:             cron.New(),
		Log:              &logger.LogrusAdapter{Logger: logrus.New()},
		taskCronEntryIDs: make(map[string]cron.EntryID),
		restyClient:      resty.New(),
		runBatches:       make(map[string]context.CancelFunc),
		baseCtx:          ctx,
		baseCancel:       cancel,
	}
}

func TestRegisterTaskJob(t *testing.T) {
	cs := newTestScheduler()
	task := &models.Task{ID: "t1", Type: "http", Schedule: "* * * * *", Method: "GET", Endpoint: "http://example.com"}

	if err := RegisterTaskJob(cs, task); err != nil {
		t.Fatalf("RegisterTaskJob returned error: %v", err)
	}
	if _, exists := cs.taskCronEntryIDs[task.ID]; !exists {
		t.Fatal("expected task to be registered in taskCronEntryIDs")
	}

	UnregisterTaskJob(cs, task.ID)
	if _, exists := cs.taskCronEntryIDs[task.ID]; exists {
		t.Fatal("expected task to be removed from taskCronEntryIDs after Unregister")
	}
}

func TestRegisterTaskJobInvalidSchedule(t *testing.T) {
	cs := newTestScheduler()
	task := &models.Task{ID: "t2", Type: "http", Schedule: "not-a-cron-expr", Method: "GET", Endpoint: "http://example.com"}

	if err := RegisterTaskJob(cs, task); err == nil {
		t.Fatal("expected error for invalid cron expression")
	}
	if _, exists := cs.taskCronEntryIDs[task.ID]; exists {
		t.Fatal("expected no map entry for a task that failed to register")
	}
}

func TestRegisterTaskJobUnknownType(t *testing.T) {
	cs := newTestScheduler()
	task := &models.Task{ID: "t3", Type: "shell", Schedule: "* * * * *"}

	if err := RegisterTaskJob(cs, task); err == nil {
		t.Fatal("expected error for unknown task type")
	}
	if _, exists := cs.taskCronEntryIDs[task.ID]; exists {
		t.Fatal("expected no map entry for an unknown task type")
	}
}

func TestRunTaskNowUnknownType(t *testing.T) {
	cs := newTestScheduler()
	task := &models.Task{ID: "t4", Type: "shell"}

	if err := RunTaskNow(cs, task, 1, 100); err == nil {
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

	if err := RunTaskNow(cs, task, 3, 20); err != nil {
		t.Fatalf("RunTaskNow returned error: %v", err)
	}
	if err := RunTaskNow(cs, task, 3, 20); !errors.Is(err, ErrBatchAlreadyRunning) {
		t.Fatalf("expected ErrBatchAlreadyRunning for concurrent batch, got %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for hits.Load() < 3 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if got := hits.Load(); got != 3 {
		t.Fatalf("expected 3 requests, got %d", got)
	}

	// 批次已跑完，registry 應已清空
	if err := CancelTaskRun(cs, task.ID); !errors.Is(err, ErrBatchNotFound) {
		t.Fatalf("expected ErrBatchNotFound after batch finished, got %v", err)
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

	if err := RunTaskNow(cs, task, 10, 200); err != nil {
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

	log := &logger.LogrusAdapter{Logger: logrus.New()}
	task := &models.Task{ID: "http4xx", Type: "http", Method: "GET", Endpoint: srv.URL, RetryCount: 2, RetryDelayMs: 0}

	RunRequestHttp(log, nil, resty.New(), task, models.SOURCE_CRON)(context.Background())

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

	log := &logger.LogrusAdapter{Logger: logrus.New()}
	task := &models.Task{ID: "http5xx", Type: "http", Method: "GET", Endpoint: srv.URL, RetryCount: 2, RetryDelayMs: 0}

	RunRequestHttp(log, nil, resty.New(), task, models.SOURCE_CRON)(context.Background())

	if got := hits.Load(); got != 3 {
		t.Fatalf("expected 3 requests (1 + 2 retries) for a retryable 5xx, got %d", got)
	}
}

func TestRegisterTaskJobReplacesOldEntry(t *testing.T) {
	cs := newTestScheduler()
	task := &models.Task{ID: "t5", Type: "http", Schedule: "* * * * *", Method: "GET", Endpoint: "http://example.com"}

	if err := RegisterTaskJob(cs, task); err != nil {
		t.Fatalf("first RegisterTaskJob returned error: %v", err)
	}
	if err := RegisterTaskJob(cs, task); err != nil {
		t.Fatalf("second RegisterTaskJob returned error: %v", err)
	}

	if got := len(cs.Cron.Entries()); got != 1 {
		t.Fatalf("expected exactly 1 cron entry after re-registering the same task, got %d", got)
	}
	if got := len(cs.taskCronEntryIDs); got != 1 {
		t.Fatalf("expected exactly 1 map entry after re-registering the same task, got %d", got)
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
		status, _, _, retryable := doHTTPAttempt(context.Background(), resty.New(), task, nil)
		srv.Close()
		if status != c.wantStatus || retryable != c.wantRetryable {
			t.Errorf("spec=%q code=%d: got (%s, %v), want (%s, %v)", c.spec, c.code, status, retryable, c.wantStatus, c.wantRetryable)
		}
	}
}
