package scheduler

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/q90016200/scheduler-go/internal/models"
	"github.com/q90016200/scheduler-go/pkg/logger"
	"github.com/robfig/cron/v3"
	"time"
)

func AddJob(cs *CronScheduler, id string, task *models.Task) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	var entryID cron.EntryID
	var err error

	switch task.Type {
	case "http":
		entryID, err = cs.Cron.AddFunc(task.Schedule, RunRequestHttp(cs.restyClient, task))
		if err != nil {
			cs.Log.Error(err, fmt.Sprintf("failed to RunHttpGet job: %+v", task))
		}

		//case "rpc":
		//case "shell":
	}

	// 保存任務 ID
	cs.jobIDs[task.ID] = entryID

	return nil
}

func RemoveJob(cs *CronScheduler, id string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	entryID, exists := cs.jobIDs[id]
	if exists {
		cs.Cron.Remove(entryID)
		cs.Log.Info("Removed job: %s", id)
	}

	return nil
}

func RunRequestHttp(client *resty.Client, task *models.Task) func() {
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
			logger.Logger.Infof("task: %+v \n RunRequestHttp fail:%v", task, err)
		}

		//if resp.StatusCode() != http.StatusOK {
		//	logger.Logger.Infof("task: %+v \n RunRequestHttp fail:%v", task, err)
		//}

		logger.Logger.Infof("task: %+v \n RunRequestHttp resp.StatusCode:%v resp:%v", task, resp.StatusCode(), string(resp.Body()))

	}
}
