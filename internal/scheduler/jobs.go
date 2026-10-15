package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/dynamic/grpcdynamic"
	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/q90016200/scheduler-go/internal/models"
	"github.com/q90016200/scheduler-go/internal/notify"
	"github.com/q90016200/scheduler-go/internal/ssrf"
	"github.com/q90016200/scheduler-go/pkg/utils"
	"github.com/rs/xid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SyncTask 讓任務的 cron 排程與 DB 目前的狀態一致：enabled 就（重新）註冊，
// 不存在或非 enabled 就移除既有 entry。所有會改動任務狀態的 mutator 都在 DB commit 之後呼叫它，
// 每分鐘 poller 也用它補齊／清掉與 DB 不一致的 entry，把「改完狀態要同步排程」收斂在這一處。
//
// 讀 DB 必須在 cs.mu 內（race guard）：mutator 一律先 commit 再 Sync，而 Sync 持鎖後才讀 DB，
// 所以鎖內讀到的一定是不早於該次 commit 的最新狀態；若在鎖外先讀、再拿鎖套用，
// 讀與套用之間別的請求可能已把任務停用／刪除並完成 Sync，這裡就會用舊快照把已停用的任務復活。
// 代價：NextRunAts 共用 cs.mu，可能要排在一次 DB round trip 之後。
// 查 DB 失敗（非 RecordNotFound）直接回錯、不動既有 entry。
func SyncTask(cs *CronScheduler, taskID string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	var task models.Task
	err := cs.db.First(&task, "id = ?", taskID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return cs.applyLocked(taskID, nil)
	}
	if err != nil {
		return err
	}
	return cs.applyLocked(taskID, &task)
}

// SyncTasks 是 SyncTask 的批次版：持 cs.mu 一次、一條 SELECT ... IN 讀回整批任務，再逐筆 applyLocked
// （DB 找不到的 id 傳 nil，等同任務已不存在）。讀 DB 仍在鎖內，race guard 與 SyncTask 相同。
// 個別任務 apply 失敗不中斷其他筆，錯誤以 errors.Join 合併回傳；整批查 DB 失敗則直接回錯、不動既有 entry。
func SyncTasks(cs *CronScheduler, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	cs.mu.Lock()
	defer cs.mu.Unlock()

	var tasks []models.Task
	if err := cs.db.Where("id IN ?", ids).Find(&tasks).Error; err != nil {
		return err
	}
	byID := make(map[string]*models.Task, len(tasks))
	for i := range tasks {
		byID[tasks[i].ID] = &tasks[i]
	}

	var errs []error
	for _, id := range ids {
		if err := cs.applyLocked(id, byID[id]); err != nil {
			errs = append(errs, fmt.Errorf("task id: %s: %w", id, err))
		}
	}
	return errors.Join(errs...)
}

// applyLocked 依 task 目前狀態調整 cron entry（task 為 nil 代表任務已不存在），呼叫端必須已持有 cs.mu。
// 不碰 DB，方便單元測試。註冊失敗時一併移除舊 entry，避免 DB 已是新設定、記憶體卻還跑著舊排程。
func (cs *CronScheduler) applyLocked(taskID string, task *models.Task) error {
	if task != nil && task.Status == models.STATUS_ENABLED {
		if err := cs.registerTaskJobLocked(task); err != nil {
			cs.removeLocked(taskID)
			return err
		}
		return nil
	}
	cs.removeLocked(taskID)
	return nil
}

// removeLocked 移除任務既有的 cron entry（若有），呼叫端必須已持有 cs.mu
func (cs *CronScheduler) removeLocked(id string) {
	if entryID, exists := cs.taskCronEntryIDs[id]; exists {
		cs.Cron.Remove(entryID)
		delete(cs.taskCronEntryIDs, id)
		cs.Log.Info(fmt.Sprintf("已移除 task %s 的 cron job", id))
	}
}

// registerTaskJobLocked 實際註冊 cron job，呼叫端必須已持有 cs.mu。
func (cs *CronScheduler) registerTaskJobLocked(task *models.Task) error {
	runOnce, err := cs.runnerFor(task, models.SOURCE_CRON, "")
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

// maxRunBatches 全站同時進行中的「立即執行」批次上限（跨任務加總），避免大量手動觸發把目標服務或本機打爆
const maxRunBatches = 10

// ErrTooManyBatches 全站進行中的「立即執行」批次已達 maxRunBatches
var ErrTooManyBatches = errors.New("too many running batches")

// ErrBatchNotFound CancelTaskRun 找不到該任務目前進行中的批次
var ErrBatchNotFound = errors.New("no running batch for task")

// attempter 單次嘗試的執行函式：依 task.Type 在 attempters 查表選用。
// 回傳的 retryable 決定 runWithRetry 要不要再試一次。
type attempter func(cs *CronScheduler, ctx context.Context, task *models.Task, headers map[string]string) (status, output, errMsg string, retryable bool)

// attempters 各 task.Type 對應的單次嘗試實作；新增 type 只要在這裡加一列（再配合 validator 的 oneof）
var attempters = map[string]attempter{
	"http": (*CronScheduler).doHTTPAttempt,
	"rpc":  (*CronScheduler).doRPCAttempt,
}

// runnerFor 依 task.Type 回傳對應的單次執行 closure（含重試、寫紀錄、連續失敗計數）；
// 未知 type 立刻回錯，讓 RunTaskNow 在開 goroutine 前就能讓呼叫端知道參數不對，而不是靜默不執行。
// batchID 是這個 closure 寫出的每筆 TaskExecutionLog 所屬的「立即執行」批次 id，cron 來源傳空字串。
func (cs *CronScheduler) runnerFor(task *models.Task, source, batchID string) (func(ctx context.Context), error) {
	attempt, ok := attempters[task.Type]
	if !ok {
		return nil, fmt.Errorf("unknown task type: %s", task.Type)
	}
	headers := cs.parseHeaders(task.Headers, task.ID)
	return func(parentCtx context.Context) {
		cs.runWithRetry(parentCtx, task, source, batchID, func(ctx context.Context) (string, string, string, bool) {
			return attempt(cs, ctx, task, headers)
		})
	}, nil
}

// RunTaskNow 立即執行任務，不經過 cron 排程，也不受 SkipIfStillRunning 影響（可能與排程執行重疊）。
// 非同步執行 repeat 次、每次間隔 intervalMs 毫秒，呼叫後立即回傳該批次的 batchID；每次執行沿用既有的 logExecution 邏輯，
// 且寫出的每筆紀錄都帶這個 batchID，呼叫端可用 IsBatchRunning + batch_id 查詢批次進度，不必自己推算哪些紀錄屬於這次。
// 中途某次執行失敗不會中斷剩下的次數。同一任務同時只允許一個進行中的批次。
// 手動觸發（source=manual）不計入 ConsecutiveFailures、也不會發失敗通知，避免使用者測試時誤觸發。
func RunTaskNow(cs *CronScheduler, task *models.Task, repeat int, intervalMs int) (batchID string, err error) {
	batchID = xid.New().String()
	runOnce, err := cs.runnerFor(task, models.SOURCE_MANUAL, batchID)
	if err != nil {
		return "", err
	}

	cs.runBatchesMu.Lock()
	// 已進入關機就不再開新批次；StopCronScheduler 取消 baseCtx 後會拿一次 runBatchesMu 當屏障，
	// 所以 batchWG.Add 一定發生在 Wait 之前
	if cs.baseCtx.Err() != nil {
		cs.runBatchesMu.Unlock()
		return "", errSchedulerClosed
	}
	if _, exists := cs.runBatches[task.ID]; exists {
		cs.runBatchesMu.Unlock()
		return "", ErrBatchAlreadyRunning
	}
	if len(cs.runBatches) >= maxRunBatches {
		cs.runBatchesMu.Unlock()
		return "", ErrTooManyBatches
	}
	ctx, cancel := context.WithCancel(cs.baseCtx)
	cs.runBatches[task.ID] = runBatch{id: batchID, cancel: cancel}
	cs.batchWG.Add(1)
	cs.runBatchesMu.Unlock()

	go func() {
		defer cs.batchWG.Done()
		// 順序保證：logExecution 是同步寫入，runOnce 回傳時該次的紀錄都已 commit；
		// 這個 defer 一定在迴圈（含所有 runOnce）結束之後才執行，所以「從 runBatches 消失」
		// 必定晚於「最後一筆紀錄寫入完成」。IsBatchRunning 回 false 之後再查 logs 就保證完整。
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
	return batchID, nil
}

// IsBatchRunning 該任務目前進行中的批次是否就是 batchID。必須同時比對 taskID 與 batchID：
// 只看 taskID 的話，舊批次已結束、同任務的新批次又開始時，舊批次會被誤判為仍在跑。
func IsBatchRunning(cs *CronScheduler, taskID, batchID string) bool {
	cs.runBatchesMu.Lock()
	defer cs.runBatchesMu.Unlock()
	b, exists := cs.runBatches[taskID]
	return exists && b.id == batchID
}

// CancelTaskRun 取消任務目前進行中的「立即執行」重複批次，只阻止尚未發出的下一次執行
func CancelTaskRun(cs *CronScheduler, taskID string) error {
	cs.runBatchesMu.Lock()
	b, exists := cs.runBatches[taskID]
	cs.runBatchesMu.Unlock()
	if !exists {
		return ErrBatchNotFound
	}
	b.cancel()
	return nil
}

// taskTimeoutSeconds 沒設定（含舊資料遷移前的 0 值）時沿用原本寫死的 10 秒
func taskTimeoutSeconds(task *models.Task) int {
	if task.TimeoutSeconds <= 0 {
		return models.DefaultTimeoutSeconds
	}
	return task.TimeoutSeconds
}

// parseHeaders 將 Task.Headers 這個自由 JSON map 字串解析成 key-value；
// 不是合法的 key-value JSON 就忽略（不阻斷任務執行），只記一筆 log
func (cs *CronScheduler) parseHeaders(raw string, taskID string) map[string]string {
	if raw == "" || raw == "{}" {
		return nil
	}
	var headers map[string]string
	if err := json.Unmarshal([]byte(raw), &headers); err != nil {
		cs.Log.Info(fmt.Sprintf("task ID: %s, headers 不是合法的 key-value JSON，忽略: %v", taskID, err))
		return nil
	}
	return headers
}

// runWithRetry 驅動「最多 1+RetryCount 次」的執行迴圈：每次嘗試各寫一筆 TaskExecutionLog
// （attemptNumber 從 1 起算），只有最後一次嘗試（次數用完，或這次錯誤不可重試）isFinalAttempt 才是 true；
// 兩次嘗試之間依 RetryDelayMs 停頓。每次嘗試的逾時掛在傳入的 parentCtx 底下，parentCtx 被取消
// （立即執行按下取消批次，或排程關機）會讓正在進行中的那次呼叫也立刻中止，不必等滿逾時或重試跑完。
// 最終結果只在 isFinalAttempt 那一刻才更新 ConsecutiveFailures / 觸發失敗通知。
func (cs *CronScheduler) runWithRetry(parentCtx context.Context, task *models.Task, source, batchID string, attemptFn func(ctx context.Context) (status, output, errMsg string, retryable bool)) {
	attempts := 1 + task.RetryCount
	timeout := time.Duration(taskTimeoutSeconds(task)) * time.Second

	for attempt := 1; attempt <= attempts; attempt++ {
		startedAt := time.Now().Unix()
		ctx, cancel := context.WithTimeout(parentCtx, timeout)
		status, output, errMsg, retryable := attemptFn(ctx)
		cancel()

		// parentCtx 已取消（取消批次或關機）：這次嘗試的失敗多半是被中止造成的，重試只會用已取消的 ctx 白跑，
		// 所以直接把這次當最後一筆寫紀錄，讓每次 run 都有一筆 final，不論 RetryDelayMs 是否為 0
		cancelled := parentCtx.Err() != nil
		isFinal := attempt == attempts || !retryable || cancelled
		logID := cs.logExecution(task.ID, startedAt, status, output, errMsg, source, batchID, attempt, isFinal)

		if isFinal {
			// 取消不是目標本身的失敗：被中止的失敗不計入連續失敗數，否則關機會誤累加並可能誤發通知；
			// 取消前已成功的那次照常清零
			if !cancelled || status == models.EXEC_STATUS_SUCCESS {
				cs.updateFailureStreakAndNotify(task, source, status, errMsg)
			}
			return
		}
		if task.RetryDelayMs > 0 {
			select {
			case <-parentCtx.Done():
				// 重試等待中被取消：剛寫的那筆就是這次觸發的最後一筆，補標 final，
				// 否則這次觸發永遠沒有 final 列（前端完成數會少算）。同樣不計入連續失敗數。
				cs.markFinal(task.ID, logID)
				return
			case <-time.After(time.Duration(task.RetryDelayMs) * time.Millisecond):
			}
		}
	}
}

// updateFailureStreakAndNotify 只在 cron 來源的最終嘗試後才更新；手動立即執行（source=manual）完全不動這個欄位。
// 成功清零 ConsecutiveFailures；失敗則 +1，「剛好」打到 NotifyFailThreshold 那一刻才發一次通知，
// 之後即使繼續失敗也靜音，直到成功一次清零重新累積，避免同一任務炸一堆重複通知。
func (cs *CronScheduler) updateFailureStreakAndNotify(task *models.Task, source, status, errMsg string) {
	db := cs.db
	if db == nil || source != models.SOURCE_CRON {
		return
	}

	if status == models.EXEC_STATUS_SUCCESS {
		// 已是 0 就不寫，避免每次成功都對同一列取寫鎖
		if err := db.Model(&models.Task{}).Where("id = ? AND consecutive_failures <> 0", task.ID).Update("consecutive_failures", 0).Error; err != nil {
			cs.Log.Error(err, fmt.Sprintf("task ID: %s, 清零連續失敗數失敗", task.ID))
		}
		return
	}

	// UPDATE ... RETURNING 一條敘述同時累加並取回新值，不必再 SELECT 重讀（兩條之間別的寫入也不會讓值錯位）
	var updated models.Task
	if err := db.Model(&updated).
		Clauses(clause.Returning{Columns: []clause.Column{{Name: "consecutive_failures"}}}).
		Where("id = ?", task.ID).
		UpdateColumn("consecutive_failures", gorm.Expr("consecutive_failures + 1")).Error; err != nil {
		cs.Log.Error(err, fmt.Sprintf("task ID: %s, 累加連續失敗數失敗", task.ID))
		return
	}

	if task.NotifyFailThreshold <= 0 {
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
func (cs *CronScheduler) doHTTPAttempt(ctx context.Context, task *models.Task, headers map[string]string) (status, output, errMsg string, retryable bool) {
	req := cs.restyClient.R().SetContext(ctx)
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

// errSchedulerClosed 排程器已 Stop 後不再建立新的 gRPC 連線（關機順序是先停 scheduler 再 srv.Shutdown，
// 期間仍可能有 HTTP 請求觸發 RunTaskNow）
var errSchedulerClosed = errors.New("scheduler closed")

// rpcConn 取得（或建立）endpoint 對應的快取 gRPC 連線。grpc.NewClient 是 lazy 且會自動重連，
// 所以連線不會因暫時性錯誤被丟棄，快取的 conn 會自癒；只有 Stop 時才統一關閉。
// ponytail: conn 不淘汰，數量上限 = 不同 endpoint 數；task 的 endpoint 被改掉後舊 conn 會留到程序結束。
// 若 endpoint 流動性變大，再加 idle 淘汰或引用計數。
func (cs *CronScheduler) rpcConn(endpoint string) (*grpc.ClientConn, error) {
	cs.rpcMu.Lock()
	defer cs.rpcMu.Unlock()

	if cs.rpcClosed {
		return nil, errSchedulerClosed
	}
	if conn, ok := cs.rpcConns[endpoint]; ok {
		return conn, nil
	}
	conn, err := grpc.NewClient(endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			return ssrf.SafeDialContext(ctx, "tcp", addr)
		}),
	)
	if err != nil {
		return nil, err
	}
	cs.rpcConns[endpoint] = conn
	return conn, nil
}

// closeRPCConns 關閉所有快取的 gRPC 連線並擋掉之後的 rpcConn，供 StopCronScheduler 使用
func (cs *CronScheduler) closeRPCConns() {
	cs.rpcMu.Lock()
	defer cs.rpcMu.Unlock()

	cs.rpcClosed = true
	for endpoint, conn := range cs.rpcConns {
		conn.Close()
		delete(cs.rpcConns, endpoint)
	}
	cs.rpcMethods = make(map[string]*desc.MethodDescriptor)
}

// rpcMethodKey MethodDescriptor 快取的 key
func rpcMethodKey(task *models.Task) string {
	return task.Endpoint + "|" + task.Method
}

// resolveRPCMethod 先查 MethodDescriptor 快取，未命中才用本次 attempt 的 ctx（已帶 headers 的 outgoing metadata）
// 做一次 reflection；取得後 Reset reflection client 再存入。method 不存在（nil）不快取。
// 回傳的 retryable 沿用原分類：reflection 失敗可重試，method not found 不可重試。
func (cs *CronScheduler) resolveRPCMethod(ctx context.Context, conn *grpc.ClientConn, task *models.Task, serviceName, methodName string) (md *desc.MethodDescriptor, errMsg string, retryable bool) {
	key := rpcMethodKey(task)
	cs.rpcMu.Lock()
	md = cs.rpcMethods[key]
	cs.rpcMu.Unlock()
	if md != nil {
		return md, "", false
	}

	refClient := grpcreflect.NewClientAuto(ctx, conn)
	svcDesc, err := refClient.ResolveService(serviceName)
	refClient.Reset()
	if err != nil {
		return nil, err.Error(), true
	}
	md = svcDesc.FindMethodByName(methodName)
	if md == nil {
		return nil, fmt.Sprintf("method not found: %s", methodName), false
	}

	cs.rpcMu.Lock()
	if !cs.rpcClosed {
		cs.rpcMethods[key] = md
	}
	cs.rpcMu.Unlock()
	return md, "", false
}

// doRPCAttempt 單次 gRPC 呼叫：dial／server reflection／invoke 失敗視為可重試（多半是網路/暫時性問題）；
// method 格式錯誤、method 不存在、payload 不是合法 JSON 視為不可重試（重試也不會有不同結果）。
// 透過 gRPC Server Reflection 動態呼叫，不需要預先產生的 pb.go stub：Endpoint 為 "host:port"，
// Method 為 "package.Service/MethodName"，Payload 為要灌入 request 訊息的 JSON 字串。
// 連線與 MethodDescriptor 都有快取（見 rpcConn / resolveRPCMethod），invoke 失敗時會丟掉該 descriptor 快取，
// 因為對方 schema 可能已變，下次重新 resolve。
func (cs *CronScheduler) doRPCAttempt(ctx context.Context, task *models.Task, headers map[string]string) (status, output, errMsg string, retryable bool) {
	serviceName, methodName, err := utils.SplitRpcMethod(task.Method)
	if err != nil {
		return models.EXEC_STATUS_FAILED, "", err.Error(), false
	}

	conn, err := cs.rpcConn(task.Endpoint)
	if err != nil {
		return models.EXEC_STATUS_FAILED, "", err.Error(), true
	}

	if len(headers) > 0 {
		ctx = metadata.NewOutgoingContext(ctx, metadata.New(headers))
	}

	methodDesc, msg, retry := cs.resolveRPCMethod(ctx, conn, task, serviceName, methodName)
	if methodDesc == nil {
		return models.EXEC_STATUS_FAILED, "", msg, retry
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
		cs.rpcMu.Lock()
		delete(cs.rpcMethods, rpcMethodKey(task))
		cs.rpcMu.Unlock()
		return models.EXEC_STATUS_FAILED, "", err.Error(), true
	}

	respJSON, err := dynamic.AsDynamicMessage(respMsg)
	if err != nil {
		return models.EXEC_STATUS_SUCCESS, fmt.Sprintf("%+v", respMsg), "", false
	}
	respBytes, _ := respJSON.MarshalJSON()
	return models.EXEC_STATUS_SUCCESS, string(respBytes), "", false
}

// logExecution 寫入一筆任務執行紀錄，寫入失敗只記 log，不影響該 task 的 cron 排程本身
// ponytail: 目前是同步寫入，若排程量大到 DB 寫入變瓶頸，再改成非同步/批次寫入
// 回傳寫入的紀錄 id（未寫入時為空字串），供 runWithRetry 在重試等待中被取消時補標 final。
func (cs *CronScheduler) logExecution(taskID string, startedAt int64, status, output, errMsg, source, batchID string, attemptNumber int, isFinalAttempt bool) string {
	if cs.db == nil {
		return ""
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
		BatchID:        batchID,
		AttemptNumber:  attemptNumber,
		IsFinalAttempt: isFinalAttempt,
	}
	if err := cs.db.Create(entry).Error; err != nil {
		cs.Log.Error(err, fmt.Sprintf("task ID: %s, 寫入執行紀錄失敗", taskID))
		return ""
	}
	return entry.ID
}

// markFinal 把已寫入的紀錄補標為該次觸發的最後一筆（重試等待中被取消時用）
func (cs *CronScheduler) markFinal(taskID, logID string) {
	if cs.db == nil || logID == "" {
		return
	}
	if err := cs.db.Model(&models.TaskExecutionLog{}).Where("id = ?", logID).Update("is_final_attempt", true).Error; err != nil {
		cs.Log.Error(err, fmt.Sprintf("task ID: %s, 補標最後一筆執行紀錄失敗", taskID))
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
