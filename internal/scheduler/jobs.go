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
	"github.com/q90016200/scheduler-go/pkg/logger"
	"github.com/q90016200/scheduler-go/pkg/utils"
	"github.com/robfig/cron/v3"
	"github.com/rs/xid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/gorm"
)

func RegisterTaskJob(cs *CronScheduler, id string, task *models.Task) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	var entryID cron.EntryID
	var err error

	switch task.Type {
	case "http":
		entryID, err = cs.Cron.AddFunc(task.Schedule, RunRequestHttp(cs.Log, cs.db, cs.restyClient, task))
	case "rpc":
		entryID, err = cs.Cron.AddFunc(task.Schedule, RunRequestRpc(cs.Log, cs.db, task))
	default:
		err = fmt.Errorf("unknown task type: %s", task.Type)
	}
	if err != nil {
		cs.Log.Error(err, fmt.Sprintf("failed to register cron job: %+v", task))
		return err
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
func jobFor(cs *CronScheduler, task *models.Task) (func(), error) {
	switch task.Type {
	case "http":
		return RunRequestHttp(cs.Log, cs.db, cs.restyClient, task), nil
	case "rpc":
		return RunRequestRpc(cs.Log, cs.db, task), nil
	default:
		return nil, fmt.Errorf("unknown task type: %s", task.Type)
	}
}

// RunTaskNow 立即執行任務，不經過 cron 排程，也不受 SkipIfStillRunning 影響（可能與排程執行重疊）。
// 非同步執行 repeat 次、每次間隔 intervalMs 毫秒，呼叫後立即回傳；每次執行沿用既有的 logExecution 邏輯。
// 中途某次執行失敗不會中斷剩下的次數。同一任務同時只允許一個進行中的批次。
func RunTaskNow(cs *CronScheduler, task *models.Task, repeat int, intervalMs int) error {
	runOnce, err := jobFor(cs, task)
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
			runOnce()
			if i == repeat-1 {
				return
			}
			// ponytail: 取消只在兩次呼叫之間生效，進行中的那一發仍會跑完各自的 10s timeout；
			// 要中斷進行中的請求需讓 RunRequestHttp/RunRequestRpc 收 ctx，範圍較大，先不做
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

func UnregisterTaskJob(cs *CronScheduler, id string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	entryID, exists := cs.taskCronEntryIDs[id]
	if exists {
		cs.Cron.Remove(entryID)
		delete(cs.taskCronEntryIDs, id)
		cs.Log.Info(fmt.Sprintf("已移除 task %s 的 cron job", id))
	}

	return nil
}

func RunRequestHttp(log *logger.LogrusAdapter, db *gorm.DB, client *resty.Client, task *models.Task) func() {
	return func() {
		startedAt := time.Now().Unix()

		var resp *resty.Response
		var err error
		switch task.Method {
		case "GET":
			resp, err = client.R().Get(task.Endpoint)
		case "POST":
			resp, err = client.R().SetBody(task.Payload).Post(task.Endpoint)
		case "PUT":
			resp, err = client.R().SetBody(task.Payload).Put(task.Endpoint)
		case "DELETE":
			resp, err = client.R().SetBody(task.Payload).Delete(task.Endpoint)
		case "PATCH":
			resp, err = client.R().SetBody(task.Payload).Patch(task.Endpoint)
		default:
			errMsg := fmt.Sprintf("unsupported http method: %s", task.Method)
			log.Info(fmt.Sprintf("task: %+v \n RunRequestHttp %s", task, errMsg))
			logExecution(db, log, task.ID, startedAt, models.EXEC_STATUS_FAILED, "", errMsg)
			return
		}

		if err != nil {
			log.Info(fmt.Sprintf("task: %+v \n RunRequestHttp fail:%v", task, task))
			logExecution(db, log, task.ID, startedAt, models.EXEC_STATUS_FAILED, "", err.Error())
			return
		}

		log.Info(fmt.Sprintf("task: %+v \n RunRequestHttp resp.StatusCode:%v resp:%v", task, resp.StatusCode(), truncate(string(resp.Body()), maxLogOutputLen)))
		logExecution(db, log, task.ID, startedAt, models.EXEC_STATUS_SUCCESS, string(resp.Body()), "")
	}
}

// RunRequestRpc 透過 gRPC Server Reflection 動態呼叫任務設定的 service/method，
// 不需要預先產生的 pb.go stub：Endpoint 為 "host:port"，Method 為 "package.Service/MethodName"，
// Payload 為要灌入 request 訊息的 JSON 字串。
func RunRequestRpc(log *logger.LogrusAdapter, db *gorm.DB, task *models.Task) func() {
	return func() {
		startedAt := time.Now().Unix()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		serviceName, methodName, err := utils.SplitRpcMethod(task.Method)
		if err != nil {
			log.Info(fmt.Sprintf("task: %+v \n RunRequestRpc invalid method: %v", task, err))
			logExecution(db, log, task.ID, startedAt, models.EXEC_STATUS_FAILED, "", err.Error())
			return
		}

		conn, err := grpc.NewClient(task.Endpoint,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
				return utils.SafeDialContext(ctx, "tcp", addr)
			}),
		)
		if err != nil {
			log.Info(fmt.Sprintf("task: %+v \n RunRequestRpc dial fail: %v", task, err))
			logExecution(db, log, task.ID, startedAt, models.EXEC_STATUS_FAILED, "", err.Error())
			return
		}
		defer conn.Close()

		refClient := grpcreflect.NewClientAuto(ctx, conn)
		defer refClient.Reset()

		svcDesc, err := refClient.ResolveService(serviceName)
		if err != nil {
			log.Info(fmt.Sprintf("task: %+v \n RunRequestRpc resolve service fail: %v", task, err))
			logExecution(db, log, task.ID, startedAt, models.EXEC_STATUS_FAILED, "", err.Error())
			return
		}

		methodDesc := svcDesc.FindMethodByName(methodName)
		if methodDesc == nil {
			errMsg := fmt.Sprintf("method not found: %s", methodName)
			log.Info(fmt.Sprintf("task: %+v \n RunRequestRpc %s", task, errMsg))
			logExecution(db, log, task.ID, startedAt, models.EXEC_STATUS_FAILED, "", errMsg)
			return
		}

		reqMsg := dynamic.NewMessage(methodDesc.GetInputType())
		if task.Payload != "" {
			if err := reqMsg.UnmarshalJSON([]byte(task.Payload)); err != nil {
				log.Info(fmt.Sprintf("task: %+v \n RunRequestRpc unmarshal payload fail: %v", task, err))
				logExecution(db, log, task.ID, startedAt, models.EXEC_STATUS_FAILED, "", err.Error())
				return
			}
		}

		stub := grpcdynamic.NewStub(conn)
		respMsg, err := stub.InvokeRpc(ctx, methodDesc, reqMsg)
		if err != nil {
			log.Info(fmt.Sprintf("task: %+v \n RunRequestRpc invoke fail: %v", task, err))
			logExecution(db, log, task.ID, startedAt, models.EXEC_STATUS_FAILED, "", err.Error())
			return
		}

		respJSON, err := dynamic.AsDynamicMessage(respMsg)
		if err != nil {
			log.Info(fmt.Sprintf("task: %+v \n RunRequestRpc resp:%+v", task, respMsg))
			logExecution(db, log, task.ID, startedAt, models.EXEC_STATUS_SUCCESS, fmt.Sprintf("%+v", respMsg), "")
			return
		}
		respBytes, _ := respJSON.MarshalJSON()
		log.Info(fmt.Sprintf("task: %+v \n RunRequestRpc resp:%v", task, truncate(string(respBytes), maxLogOutputLen)))
		logExecution(db, log, task.ID, startedAt, models.EXEC_STATUS_SUCCESS, string(respBytes), "")
	}
}

// logExecution 寫入一筆任務執行紀錄，寫入失敗只記 log，不影響該 task 的 cron 排程本身
// ponytail: 目前是同步寫入，若排程量大到 DB 寫入變瓶頸，再改成非同步/批次寫入
func logExecution(db *gorm.DB, log *logger.LogrusAdapter, taskID string, startedAt int64, status, output, errMsg string) {
	if db == nil {
		return
	}
	entry := &models.TaskExecutionLog{
		ID:         xid.New().String(),
		TaskID:     taskID,
		Status:     status,
		StartedAt:  startedAt,
		FinishedAt: time.Now().Unix(),
		Output:     toJSONColumn(output),
		ErrMessage: errMsg,
	}
	if err := db.Create(entry).Error; err != nil {
		log.Error(err, fmt.Sprintf("task ID: %s, 寫入執行紀錄失敗", taskID))
	}
}

// maxLogOutputLen 寫進 log 的回應內容長度上限，避免大回應灌爆 log 檔
const maxLogOutputLen = 2000

// truncate 將字串截斷至最多 n 個位元組
func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// toJSONColumn 確保寫進 output（Postgres jsonb 欄位）的內容一定是合法 JSON：
// http 任務的回應本來就可能是 JSON，直接原樣存；不是合法 JSON（例如純文字/HTML）就包一層 {"raw": "..."}。
func toJSONColumn(output string) string {
	output = truncate(output, maxLogOutputLen)
	if output == "" {
		return "{}"
	}
	if json.Valid([]byte(output)) {
		return output
	}
	wrapped, _ := json.Marshal(map[string]string{"raw": output})
	return string(wrapped)
}
