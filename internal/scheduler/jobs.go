package scheduler

import (
	"github.com/q90016200/scheduler-go/internal/models"
)

func AddJob(cs *CronScheduler, id string, task *models.Task) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	switch task.Type {
	case "http":

		//case "rpc":
	}

	// 保存任務 ID
	//cs.jobIDs[task.ID] = entryID

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

//func RunHttpGet() func() {
//
//}
//
//func RunHttpPost() func() {
//
//}
