package scheduler

import (
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/q90016200/scheduler-go/internal/models"
	"github.com/q90016200/scheduler-go/pkg/logger"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

func newTestScheduler() *CronScheduler {
	return &CronScheduler{
		Cron:             cron.New(),
		Log:              &logger.LogrusAdapter{Logger: logrus.New()},
		taskCronEntryIDs: make(map[string]cron.EntryID),
		restyClient:      resty.New(),
	}
}

func TestRegisterTaskJob(t *testing.T) {
	cs := newTestScheduler()
	task := &models.Task{ID: "t1", Type: "http", Schedule: "* * * * *", Method: "GET", Endpoint: "http://example.com"}

	if err := RegisterTaskJob(cs, task.ID, task); err != nil {
		t.Fatalf("RegisterTaskJob returned error: %v", err)
	}
	if _, exists := cs.taskCronEntryIDs[task.ID]; !exists {
		t.Fatal("expected task to be registered in taskCronEntryIDs")
	}

	if err := UnregisterTaskJob(cs, task.ID); err != nil {
		t.Fatalf("UnregisterTaskJob returned error: %v", err)
	}
	if _, exists := cs.taskCronEntryIDs[task.ID]; exists {
		t.Fatal("expected task to be removed from taskCronEntryIDs after Unregister")
	}
}

func TestRegisterTaskJobInvalidSchedule(t *testing.T) {
	cs := newTestScheduler()
	task := &models.Task{ID: "t2", Type: "http", Schedule: "not-a-cron-expr", Method: "GET", Endpoint: "http://example.com"}

	if err := RegisterTaskJob(cs, task.ID, task); err == nil {
		t.Fatal("expected error for invalid cron expression")
	}
	if _, exists := cs.taskCronEntryIDs[task.ID]; exists {
		t.Fatal("expected no map entry for a task that failed to register")
	}
}

func TestRegisterTaskJobUnknownType(t *testing.T) {
	cs := newTestScheduler()
	task := &models.Task{ID: "t3", Type: "shell", Schedule: "* * * * *"}

	if err := RegisterTaskJob(cs, task.ID, task); err == nil {
		t.Fatal("expected error for unknown task type")
	}
	if _, exists := cs.taskCronEntryIDs[task.ID]; exists {
		t.Fatal("expected no map entry for an unknown task type")
	}
}
