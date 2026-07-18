package scheduler

import (
	"context"
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
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func AddJob(cs *CronScheduler, id string, task *models.Task) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	var entryID cron.EntryID
	var err error

	switch task.Type {
	case "http":
		entryID, err = cs.Cron.AddFunc(task.Schedule, RunRequestHttp(cs.Log, cs.restyClient, task))
		if err != nil {
			cs.Log.Error(err, fmt.Sprintf("failed to RunHttpGet job: %+v", task))
		}

	case "rpc":
		entryID, err = cs.Cron.AddFunc(task.Schedule, RunRequestRpc(cs.Log, task))
		if err != nil {
			cs.Log.Error(err, fmt.Sprintf("failed to RunRequestRpc job: %+v", task))
		}

		//case "shell":
	}

	// 保存任務 ID
	cs.Log.Info(fmt.Sprintf("新增 job: %s", task.ID))
	cs.jobIDs[task.ID] = entryID

	return nil
}

func RemoveJob(cs *CronScheduler, id string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	entryID, exists := cs.jobIDs[id]
	if exists {
		cs.Cron.Remove(entryID)
		cs.Log.Info(fmt.Sprintf("Removed job: %s", id))
	}

	return nil
}

func RunRequestHttp(log *logger.LogrusAdapter, client *resty.Client, task *models.Task) func() {
	return func() {
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
		}

		//if resp.StatusCode() != http.StatusOK {
		//	log.Infof("task: %+v \n RunRequestHttp fail:%v", task, err)
		//}

		log.Info(fmt.Sprintf("task: %+v \n RunRequestHttp resp.StatusCode:%v resp:%v", task, resp.StatusCode(), string(resp.Body())))

	}
}

// RunRequestRpc 透過 gRPC Server Reflection 動態呼叫任務設定的 service/method，
// 不需要預先產生的 pb.go stub：Endpoint 為 "host:port"，Method 為 "package.Service/MethodName"，
// Payload 為要灌入 request 訊息的 JSON 字串。
func RunRequestRpc(log *logger.LogrusAdapter, task *models.Task) func() {
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		serviceName, methodName, err := splitRpcMethod(task.Method)
		if err != nil {
			log.Info(fmt.Sprintf("task: %+v \n RunRequestRpc invalid method: %v", task, err))
			return
		}

		conn, err := grpc.NewClient(task.Endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Info(fmt.Sprintf("task: %+v \n RunRequestRpc dial fail: %v", task, err))
			return
		}
		defer conn.Close()

		refClient := grpcreflect.NewClientAuto(ctx, conn)
		defer refClient.Reset()

		svcDesc, err := refClient.ResolveService(serviceName)
		if err != nil {
			log.Info(fmt.Sprintf("task: %+v \n RunRequestRpc resolve service fail: %v", task, err))
			return
		}

		methodDesc := svcDesc.FindMethodByName(methodName)
		if methodDesc == nil {
			log.Info(fmt.Sprintf("task: %+v \n RunRequestRpc method not found: %s", task, methodName))
			return
		}

		reqMsg := dynamic.NewMessage(methodDesc.GetInputType())
		if task.Payload != "" {
			if err := reqMsg.UnmarshalJSON([]byte(task.Payload)); err != nil {
				log.Info(fmt.Sprintf("task: %+v \n RunRequestRpc unmarshal payload fail: %v", task, err))
				return
			}
		}

		stub := grpcdynamic.NewStub(conn)
		respMsg, err := stub.InvokeRpc(ctx, methodDesc, reqMsg)
		if err != nil {
			log.Info(fmt.Sprintf("task: %+v \n RunRequestRpc invoke fail: %v", task, err))
			return
		}

		respJSON, err := dynamic.AsDynamicMessage(respMsg)
		if err != nil {
			log.Info(fmt.Sprintf("task: %+v \n RunRequestRpc resp:%+v", task, respMsg))
			return
		}
		respBytes, _ := respJSON.MarshalJSON()
		log.Info(fmt.Sprintf("task: %+v \n RunRequestRpc resp:%v", task, string(respBytes)))
	}
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
