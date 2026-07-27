package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/dynamic/grpcdynamic"
	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/q90016200/scheduler-go/internal/models"
	"github.com/q90016200/scheduler-go/pkg/logger"
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

// RunTaskNow 立即執行一次任務（不經過 cron 排程），沿用既有的執行與 logExecution 邏輯
func RunTaskNow(cs *CronScheduler, task *models.Task) error {
	switch task.Type {
	case "http":
		RunRequestHttp(cs.Log, cs.db, cs.restyClient, task)()
	case "rpc":
		RunRequestRpc(cs.Log, cs.db, task)()
	default:
		return fmt.Errorf("unknown task type: %s", task.Type)
	}
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
		client = client.SetTimeout(time.Duration(10) * time.Second)

		var resp *resty.Response
		var err error
		if task.Method == "GET" {
			resp, err = client.R().Get(task.Endpoint)
		} else if task.Method == "POST" || task.Method == "PUT" || task.Method == "DELETE" {
			req := client.R().SetBody(task.Payload)

			switch task.Method {
			case "POST":
				resp, err = req.Post(task.Endpoint)
			case "PUT":
				resp, err = req.Put(task.Endpoint)
			case "DELETE":
				resp, err = req.Delete(task.Endpoint)
			}
		}

		if err != nil {
			log.Info(fmt.Sprintf("task: %+v \n RunRequestHttp fail:%v", task, task))
			logExecution(db, log, task.ID, startedAt, models.EXEC_STATUS_FAILED, "", err.Error())
			return
		}

		//if resp.StatusCode() != http.StatusOK {
		//	log.Infof("task: %+v \n RunRequestHttp fail:%v", task, err)
		//}

		log.Info(fmt.Sprintf("task: %+v \n RunRequestHttp resp.StatusCode:%v resp:%v", task, resp.StatusCode(), string(resp.Body())))
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

		serviceName, methodName, err := splitRpcMethod(task.Method)
		if err != nil {
			log.Info(fmt.Sprintf("task: %+v \n RunRequestRpc invalid method: %v", task, err))
			logExecution(db, log, task.ID, startedAt, models.EXEC_STATUS_FAILED, "", err.Error())
			return
		}

		conn, err := grpc.NewClient(task.Endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
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
		log.Info(fmt.Sprintf("task: %+v \n RunRequestRpc resp:%v", task, string(respBytes)))
		logExecution(db, log, task.ID, startedAt, models.EXEC_STATUS_SUCCESS, string(respBytes), "")
	}
}

// logExecution 寫入一筆任務執行紀錄，寫入失敗只記 log，不影響該 task 的 cron 排程本身
// ponytail: 目前是同步寫入，若排程量大到 DB 寫入變瓶頸，再改成非同步/批次寫入
func logExecution(db *gorm.DB, log *logger.LogrusAdapter, taskID string, startedAt int64, status, output, errMsg string) {
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

// toJSONColumn 確保寫進 output（Postgres jsonb 欄位）的內容一定是合法 JSON：
// http 任務的回應本來就可能是 JSON，直接原樣存；不是合法 JSON（例如純文字/HTML）就包一層 {"raw": "..."}。
func toJSONColumn(output string) string {
	const maxOutputLen = 2000
	if len(output) > maxOutputLen {
		output = output[:maxOutputLen]
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

// splitRpcMethod 將 "package.Service/MethodName"（允許帶前導 "/"）拆成 service 全名與方法名稱。
func splitRpcMethod(method string) (serviceName string, methodName string, err error) {
	method = strings.TrimPrefix(method, "/")
	idx := strings.LastIndex(method, "/")
	if idx <= 0 || idx == len(method)-1 {
		return "", "", fmt.Errorf("method must be in the form \"package.Service/MethodName\", got %q", method)
	}
	return method[:idx], method[idx+1:], nil
}
