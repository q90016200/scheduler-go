package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/dynamic/grpcdynamic"
	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/q90016200/scheduler-go/internal/models"
	"github.com/q90016200/scheduler-go/internal/notify"
	"github.com/q90016200/scheduler-go/internal/ssrf"
	"github.com/q90016200/scheduler-go/pkg/logger"
	"github.com/q90016200/scheduler-go/pkg/utils"
	"github.com/rs/xid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"gorm.io/gorm"
)

func RegisterTaskJob(cs *CronScheduler, task *models.Task) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	return registerTaskJobLocked(cs, task)
}

// registerTaskJobLocked 實際註冊 cron job，呼叫端必須已持有 cs.mu。
func registerTaskJobLocked(cs *CronScheduler, task *models.Task) error {
	runOnce, err := jobFor(cs, task, models.SOURCE_CRON)
	if err != nil {
		cs.Log.Error(err, fmt.Sprintf("failed to register cron job: %+v", task))
		return err
	}

	entryID, err := cs.Cron.AddFunc(task.Schedule, func() { runOnce(cs.baseCtx) })
	if err != nil {
		cs.Log.Error(err, fmt.Sprintf("failed to register cron job: %+v", task))
		return err
	}

	// 若該 task 已有舊的 cron entry（例如併發 enable/update，或 CreateTask 與每分鐘 poller 競速），
	// 註冊新 entry 後要把舊的移除，否則舊 entry 永遠留在 cron 裡、同一 task 變成兩個排程同時觸發
	if oldEntryID, exists := cs.taskCronEntryIDs[task.ID]; exists {
		cs.Cron.Remove(oldEntryID)
	}

	// 保存任務 ID
	cs.Log.Info(fmt.Sprintf("已為 task %s 註冊 cron job", task.ID))
	cs.taskCronEntryIDs[task.ID] = entryID

	return nil
}

// ErrBatchAlreadyRunning 同一任務同時只允許一個進行中的「立即執行」重複批次
var ErrBatchAlreadyRunning = errors.New("task already has a running batch")

// ErrBatchNotFound CancelTaskRun 找不到該任務目前進行中的批次
var ErrBatchNotFound = errors.New("no running batch for task")

// jobFor 依 task.Type 回傳對應的單次執行 closure；未知 type 立刻回錯，
// 讓 RunTaskNow 在開 goroutine 前就能讓呼叫端知道參數不對，而不是靜默不執行。
func jobFor(cs *CronScheduler, task *models.Task, source string) (func(ctx context.Context), error) {
	switch task.Type {
	case "http":
		return RunRequestHttp(cs.Log, cs.db, cs.restyClient, task, source), nil
	case "rpc":
		return RunRequestRpc(cs.Log, cs.db, task, source), nil
	default:
		return nil, fmt.Errorf("unknown task type: %s", task.Type)
	}
}

// RunTaskNow 立即執行任務，不經過 cron 排程，也不受 SkipIfStillRunning 影響（可能與排程執行重疊）。
// 非同步執行 repeat 次、每次間隔 intervalMs 毫秒，呼叫後立即回傳；每次執行沿用既有的 logExecution 邏輯。
// 中途某次執行失敗不會中斷剩下的次數。同一任務同時只允許一個進行中的批次。
// 手動觸發（source=manual）不計入 ConsecutiveFailures、也不會發失敗通知，避免使用者測試時誤觸發。
func RunTaskNow(cs *CronScheduler, task *models.Task, repeat int, intervalMs int) error {
	runOnce, err := jobFor(cs, task, models.SOURCE_MANUAL)
	if err != nil {
		return err
	}

	cs.runBatchesMu.Lock()
	if _, exists := cs.runBatches[task.ID]; exists {
		cs.runBatchesMu.Unlock()
		return ErrBatchAlreadyRunning
	}
	ctx, cancel := context.WithCancel(cs.baseCtx)
	cs.runBatches[task.ID] = cancel
	cs.runBatchesMu.Unlock()

	go func() {
		defer func() {
			cancel()
			cs.runBatchesMu.Lock()
			delete(cs.runBatches, task.ID)
			cs.runBatchesMu.Unlock()
		}()
		for i := 0; i < repeat; i++ {
			if ctx.Err() != nil {
				return
			}
			runOnce(ctx)
			if i == repeat-1 {
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Duration(intervalMs) * time.Millisecond):
			}
		}
	}()
	return nil
}

// CancelTaskRun 取消任務目前進行中的「立即執行」重複批次，只阻止尚未發出的下一次執行
func CancelTaskRun(cs *CronScheduler, taskID string) error {
	cs.runBatchesMu.Lock()
	cancel, exists := cs.runBatches[taskID]
	cs.runBatchesMu.Unlock()
	if !exists {
		return ErrBatchNotFound
	}
	cancel()
	return nil
}

func UnregisterTaskJob(cs *CronScheduler, id string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	entryID, exists := cs.taskCronEntryIDs[id]
	if exists {
		cs.Cron.Remove(entryID)
		delete(cs.taskCronEntryIDs, id)
		cs.Log.Info(fmt.Sprintf("已移除 task %s 的 cron job", id))
	}
}

// taskTimeoutSeconds 沒設定（含舊資料遷移前的 0 值）時沿用原本寫死的 10 秒
func taskTimeoutSeconds(task *models.Task) int {
	if task.TimeoutSeconds <= 0 {
		return 10
	}
	return task.TimeoutSeconds
}

// parseHeaders 將 Task.Headers 這個自由 JSON map 字串解析成 key-value；
// 不是合法的 key-value JSON 就忽略（不阻斷任務執行），只記一筆 log
func parseHeaders(raw string, log *logger.LogrusAdapter, taskID string) map[string]string {
	if raw == "" || raw == "{}" {
		return nil
	}
	var headers map[string]string
	if err := json.Unmarshal([]byte(raw), &headers); err != nil {
		log.Info(fmt.Sprintf("task ID: %s, headers 不是合法的 key-value JSON，忽略: %v", taskID, err))
		return nil
	}
	return headers
}

// runWithRetry 驅動「最多 1+RetryCount 次」的執行迴圈：每次嘗試各寫一筆 TaskExecutionLog
// （attemptNumber 從 1 起算），只有最後一次嘗試（次數用完，或這次錯誤不可重試）isFinalAttempt 才是 true；
// 兩次嘗試之間依 RetryDelayMs 停頓。每次嘗試的逾時掛在傳入的 parentCtx 底下，parentCtx 被取消
// （立即執行按下取消批次，或排程關機）會讓正在進行中的那次呼叫也立刻中止，不必等滿逾時或重試跑完。
// 最終結果只在 isFinalAttempt 那一刻才更新 ConsecutiveFailures / 觸發失敗通知。
func runWithRetry(parentCtx context.Context, db *gorm.DB, log *logger.LogrusAdapter, task *models.Task, source string, attemptFn func(ctx context.Context) (status, output, errMsg string, retryable bool)) {
	attempts := 1 + task.RetryCount
	timeout := time.Duration(taskTimeoutSeconds(task)) * time.Second

	for attempt := 1; attempt <= attempts; attempt++ {
		startedAt := time.Now().Unix()
		ctx, cancel := context.WithTimeout(parentCtx, timeout)
		status, output, errMsg, retryable := attemptFn(ctx)
		cancel()

		isFinal := attempt == attempts || !retryable
		logExecution(db, log, task.ID, startedAt, status, output, errMsg, source, attempt, isFinal)

		if isFinal {
			updateFailureStreakAndNotify(db, log, task, source, status, errMsg)
			return
		}
		if task.RetryDelayMs > 0 {
			select {
			case <-parentCtx.Done():
				return
			case <-time.After(time.Duration(task.RetryDelayMs) * time.Millisecond):
			}
		}
	}
}

// updateFailureStreakAndNotify 只在 cron 來源的最終嘗試後才更新；手動立即執行（source=manual）完全不動這個欄位。
// 成功清零 ConsecutiveFailures；失敗則 +1，「剛好」打到 NotifyFailThreshold 那一刻才發一次通知，
// 之後即使繼續失敗也靜音，直到成功一次清零重新累積，避免同一任務炸一堆重複通知。
func updateFailureStreakAndNotify(db *gorm.DB, log *logger.LogrusAdapter, task *models.Task, source, status, errMsg string) {
	if db == nil || source != models.SOURCE_CRON {
		return
	}

	if status == models.EXEC_STATUS_SUCCESS {
		if err := db.Model(&models.Task{}).Where("id = ?", task.ID).Update("consecutive_failures", 0).Error; err != nil {
			log.Error(err, fmt.Sprintf("task ID: %s, 清零連續失敗數失敗", task.ID))
		}
		return
	}

	if err := db.Model(&models.Task{}).Where("id = ?", task.ID).
		UpdateColumn("consecutive_failures", gorm.Expr("consecutive_failures + 1")).Error; err != nil {
		log.Error(err, fmt.Sprintf("task ID: %s, 累加連續失敗數失敗", task.ID))
		return
	}

	if task.NotifyFailThreshold <= 0 {
		return
	}

	var updated models.Task
	if err := db.Select("consecutive_failures").Where("id = ?", task.ID).First(&updated).Error; err != nil {
		log.Error(err, fmt.Sprintf("task ID: %s, 讀取連續失敗數失敗", task.ID))
		return
	}

	if updated.ConsecutiveFailures == task.NotifyFailThreshold {
		notify.Dispatch(task, notify.Message{
			TaskID:     task.ID,
			TaskName:   task.Name,
			ErrMessage: errMsg,
			FailedAt:   time.Now(),
		})
	}
}

// doHTTPAttempt 單次 HTTP 呼叫：網路錯誤／timeout／5xx 視為可重試；4xx 與不支援的 method 視為不可重試
// （重試也不會有不同結果）。task.SuccessStatusCodes 有值時改以它判定成功。ctx 帶著 per-task 逾時，取代舊版寫死在 client 上的 10 秒。
func doHTTPAttempt(ctx context.Context, client *resty.Client, task *models.Task, headers map[string]string) (status, output, errMsg string, retryable bool) {
	req := client.R().SetContext(ctx)
	if len(headers) > 0 {
		req.SetHeaders(headers)
	}

	var resp *resty.Response
	var err error
	switch task.Method {
	case "GET":
		resp, err = req.Get(task.Endpoint)
	case "POST":
		resp, err = req.SetBody(task.Payload).Post(task.Endpoint)
	case "PUT":
		resp, err = req.SetBody(task.Payload).Put(task.Endpoint)
	case "DELETE":
		resp, err = req.SetBody(task.Payload).Delete(task.Endpoint)
	case "PATCH":
		resp, err = req.SetBody(task.Payload).Patch(task.Endpoint)
	default:
		return models.EXEC_STATUS_FAILED, "", fmt.Sprintf("unsupported http method: %s", task.Method), false
	}

	if err != nil {
		return models.EXEC_STATUS_FAILED, "", err.Error(), true
	}
	// 任務有自訂成功狀態碼時以它為準；不符合時仍沿用 5xx 可重試、其餘不可重試的分類
	if task.SuccessStatusCodes != "" {
		code := resp.StatusCode()
		if ok, _ := utils.MatchStatusCode(task.SuccessStatusCodes, code); ok {
			return models.EXEC_STATUS_SUCCESS, string(resp.Body()), "", false
		}
		return models.EXEC_STATUS_FAILED, string(resp.Body()), fmt.Sprintf("http %d, expected %s", code, task.SuccessStatusCodes), code >= 500
	}
	if resp.StatusCode() >= 500 {
		return models.EXEC_STATUS_FAILED, string(resp.Body()), fmt.Sprintf("http %d", resp.StatusCode()), true
	}
	if resp.StatusCode() >= 400 {
		return models.EXEC_STATUS_FAILED, string(resp.Body()), fmt.Sprintf("http %d", resp.StatusCode()), false
	}
	return models.EXEC_STATUS_SUCCESS, string(resp.Body()), "", false
}

func RunRequestHttp(log *logger.LogrusAdapter, db *gorm.DB, client *resty.Client, task *models.Task, source string) func(ctx context.Context) {
	headers := parseHeaders(task.Headers, log, task.ID)
	return func(parentCtx context.Context) {
		runWithRetry(parentCtx, db, log, task, source, func(ctx context.Context) (string, string, string, bool) {
			return doHTTPAttempt(ctx, client, task, headers)
		})
	}
}

// doRPCAttempt 單次 gRPC 呼叫：dial／server reflection／invoke 失敗視為可重試（多半是網路/暫時性問題）；
// method 格式錯誤、method 不存在、payload 不是合法 JSON 視為不可重試（重試也不會有不同結果）。
func doRPCAttempt(ctx context.Context, task *models.Task, headers map[string]string) (status, output, errMsg string, retryable bool) {
	serviceName, methodName, err := utils.SplitRpcMethod(task.Method)
	if err != nil {
		return models.EXEC_STATUS_FAILED, "", err.Error(), false
	}

	conn, err := grpc.NewClient(task.Endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			return ssrf.SafeDialContext(ctx, "tcp", addr)
		}),
	)
	if err != nil {
		return models.EXEC_STATUS_FAILED, "", err.Error(), true
	}
	defer conn.Close()

	if len(headers) > 0 {
		ctx = metadata.NewOutgoingContext(ctx, metadata.New(headers))
	}

	refClient := grpcreflect.NewClientAuto(ctx, conn)
	defer refClient.Reset()

	svcDesc, err := refClient.ResolveService(serviceName)
	if err != nil {
		return models.EXEC_STATUS_FAILED, "", err.Error(), true
	}

	methodDesc := svcDesc.FindMethodByName(methodName)
	if methodDesc == nil {
		return models.EXEC_STATUS_FAILED, "", fmt.Sprintf("method not found: %s", methodName), false
	}

	reqMsg := dynamic.NewMessage(methodDesc.GetInputType())
	if task.Payload != "" {
		if err := reqMsg.UnmarshalJSON([]byte(task.Payload)); err != nil {
			return models.EXEC_STATUS_FAILED, "", err.Error(), false
		}
	}

	stub := grpcdynamic.NewStub(conn)
	respMsg, err := stub.InvokeRpc(ctx, methodDesc, reqMsg)
	if err != nil {
		return models.EXEC_STATUS_FAILED, "", err.Error(), true
	}

	respJSON, err := dynamic.AsDynamicMessage(respMsg)
	if err != nil {
		return models.EXEC_STATUS_SUCCESS, fmt.Sprintf("%+v", respMsg), "", false
	}
	respBytes, _ := respJSON.MarshalJSON()
	return models.EXEC_STATUS_SUCCESS, string(respBytes), "", false
}

// RunRequestRpc 透過 gRPC Server Reflection 動態呼叫任務設定的 service/method，
// 不需要預先產生的 pb.go stub：Endpoint 為 "host:port"，Method 為 "package.Service/MethodName"，
// Payload 為要灌入 request 訊息的 JSON 字串。
func RunRequestRpc(log *logger.LogrusAdapter, db *gorm.DB, task *models.Task, source string) func(ctx context.Context) {
	headers := parseHeaders(task.Headers, log, task.ID)
	return func(parentCtx context.Context) {
		runWithRetry(parentCtx, db, log, task, source, func(ctx context.Context) (string, string, string, bool) {
			return doRPCAttempt(ctx, task, headers)
		})
	}
}

// logExecution 寫入一筆任務執行紀錄，寫入失敗只記 log，不影響該 task 的 cron 排程本身
// ponytail: 目前是同步寫入，若排程量大到 DB 寫入變瓶頸，再改成非同步/批次寫入
func logExecution(db *gorm.DB, log *logger.LogrusAdapter, taskID string, startedAt int64, status, output, errMsg, source string, attemptNumber int, isFinalAttempt bool) {
	if db == nil {
		return
	}
	entry := &models.TaskExecutionLog{
		ID:             xid.New().String(),
		TaskID:         taskID,
		Status:         status,
		StartedAt:      startedAt,
		FinishedAt:     time.Now().Unix(),
		Output:         toJSONColumn(output),
		ErrMessage:     errMsg,
		Source:         source,
		AttemptNumber:  attemptNumber,
		IsFinalAttempt: isFinalAttempt,
	}
	if err := db.Create(entry).Error; err != nil {
		log.Error(err, fmt.Sprintf("task ID: %s, 寫入執行紀錄失敗", taskID))
	}
}

// maxLogOutputLen 寫進 log 的回應內容長度上限，避免大回應灌爆 log 檔
const maxLogOutputLen = 2000

// toJSONColumn 確保寫進 output（Postgres jsonb 欄位）的內容一定是合法 JSON：
// http 任務的回應本來就可能是 JSON，直接原樣存；不是合法 JSON（例如純文字/HTML）就包一層 {"raw": "..."}。
func toJSONColumn(output string) string {
	if len(output) > maxLogOutputLen {
		output = output[:maxLogOutputLen]
	}
	if output == "" {
		return "{}"
	}
	if json.Valid([]byte(output)) {
		return output
	}
	wrapped, _ := json.Marshal(map[string]string{"raw": output})
	return string(wrapped)
}
