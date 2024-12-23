package scheduler

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/q90016200/scheduler-go/internal/models"
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

		if task.Method == "GET" {
			client.R().Get(task.Endpoint)
		} else if task.Method == "POST" || task.Method == "PUT" || task.Method == "DELETE" {
			req := client.R().SetBody(task.Payload)

			switch task.Method {
			case "POST":
				req.Post(task.Endpoint)
			case "PUT":
				req.Put(task.Endpoint)
			case "DELETE":
				req.Delete(task.Endpoint)
			}
		}

	}
}
