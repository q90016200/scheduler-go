package services

import (
	"fmt"
	"github.com/q90016200/scheduler-go/internal/models"
	"github.com/q90016200/scheduler-go/internal/scheduler"
	"github.com/q90016200/scheduler-go/pkg/logger"
	"github.com/rs/xid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"slices"
)

type TaskService struct {
	db            *gorm.DB
	cronScheduler *scheduler.CronScheduler
}

func NewTaskService(db *gorm.DB, cronScheduler *scheduler.CronScheduler) *TaskService {
	return &TaskService{
		cronScheduler: cronScheduler,
		db:            db,
	}
}

// listTasksSortableFields 排序欄位白名單，避免 sortField 直接拼進 SQL 造成 injection
var listTasksSortableFields = map[string]bool{
	"name":       true,
	"created_at": true,
	"status":     true,
	"schedule":   true,
}

// listTaskGroupsSortableFields 群組排序欄位白名單，理由同 listTasksSortableFields
var listTaskGroupsSortableFields = map[string]bool{
	"name":       true,
	"created_at": true,
	"status":     true,
}

// orderBy 依白名單 allowed 決定排序字串：sortField 不在白名單時回傳 fallback（空字串代表呼叫端不套用排序，
// 沿用查詢的預設順序），在白名單內則組出 "欄位 ASC/DESC"（sortOrder 為 "desc" 才是 DESC，其餘一律 ASC）
func orderBy(allowed map[string]bool, sortField, sortOrder, fallback string) string {
	if !allowed[sortField] {
		return fallback
	}
	dir := "ASC"
	if sortOrder == "desc" {
		dir = "DESC"
	}
	return sortField + " " + dir
}

// ListTasks 支持篩選、排序與分頁
func (s *TaskService) ListTasks(status, groupID, name, sortField, sortOrder string, limit, offset int) ([]models.Task, int64, error) {
	var tasks []models.Task
	query := s.db.Model(&models.Task{})

	// 篩選條件
	if status != "" {
		query = query.Where("tasks.status = ?", status)
	}
	if groupID != "" {
		query = query.Where("tasks.group_id = ?", groupID)
	}
	if name != "" {
		query = query.Where("tasks.name ILIKE ?", "%"+name+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 排序條件
	if order := orderBy(listTasksSortableFields, sortField, sortOrder, ""); order != "" {
		query = query.Order(order)
	}

	// 查詢結果
	err := query.Preload("Group").Limit(limit).Offset(offset).Find(&tasks).Error
	if err != nil {
		return nil, 0, err
	}
	s.decorateNextRunAt(tasks)
	return tasks, total, nil
}

// GetTask 取得單一任務
func (s *TaskService) GetTask(id string) (*models.Task, error) {
	var task models.Task
	if err := s.db.Preload("Group").First(&task, "id = ?", id).Error; err != nil {
		return nil, err
	}
	one := []models.Task{task}
	s.decorateNextRunAt(one)
	return &one[0], nil
}

// decorateNextRunAt 一次從記憶體中的 cron entry 查整批任務的下次執行時間裝飾進 Task；
// 未掛上 cron job（未啟用、或註冊失敗）的任務保持 nil，前端顯示「-」
func (s *TaskService) decorateNextRunAt(tasks []models.Task) {
	ids := make([]string, len(tasks))
	for i := range tasks {
		ids[i] = tasks[i].ID
	}
	nexts := scheduler.NextRunAts(s.cronScheduler, ids)
	for i := range tasks {
		if next, ok := nexts[tasks[i].ID]; ok {
			unix := next.Unix()
			tasks[i].NextRunAt = &unix
		}
	}
}

// taskFromReq 把 CreateTaskReq/UpdateTaskReq（同一型別）映射進 *models.Task，並套用
// TimeoutSeconds/Headers 的預設值（欄位留空時 GORM 不會套用 column default，須在這裡補上）。
// CreatedBy 只有 CreateTask 需要，由呼叫端在拿到回傳的 task 後自行補設。
func taskFromReq(id string, input models.CreateTaskReq) *models.Task {
	task := &models.Task{
		ID:                  id,
		GroupID:             input.GroupID,
		Name:                input.Name,
		Description:         input.Description,
		Schedule:            input.Schedule,
		Type:                input.Type,
		Endpoint:            input.EndPoint,
		Method:              input.Method,
		Payload:             input.Payload,
		Status:              input.Status,
		UpdatedBy:           input.UserId,
		RetryCount:          input.RetryCount,
		RetryDelayMs:        input.RetryDelayMs,
		TimeoutSeconds:      input.TimeoutSeconds,
		Headers:             input.Headers,
		SuccessStatusCodes:  input.SuccessStatusCodes,
		NotifyFailThreshold: input.NotifyFailThreshold,
		NotifyTelegram:      input.NotifyTelegram,
		NotifySlack:         input.NotifySlack,
	}
	if task.TimeoutSeconds == 0 {
		task.TimeoutSeconds = models.DefaultTimeoutSeconds
	}
	if task.Headers == "" {
		task.Headers = "{}"
	}
	return task
}

func (s *TaskService) CreateTask(input models.CreateTaskReq) (*models.Task, error) {
	task := taskFromReq(xid.New().String(), input)
	task.CreatedBy = input.UserId

	if err := s.db.Create(task).Error; err != nil {
		return nil, err
	}

	// 依 DB 狀態同步排程（enabled 才會註冊）
	if err := scheduler.SyncTask(s.cronScheduler, task.ID); err != nil {
		// 同步排程失敗，不留下無法排程的孤兒資料
		if delErr := s.db.Delete(&models.Task{}, "id = ?", task.ID).Error; delErr != nil {
			logger.Log.Errorf("task id: %s, 同步排程失敗後清理孤兒資料也失敗: %v", task.ID, delErr)
		}
		return nil, err
	}

	return task, nil
}

// taskUpdatableColumns UpdateTask 允許更新的欄位，排除 created_at/created_by
var taskUpdatableColumns = []string{
	"group_id", "name", "description", "schedule", "type", "endpoint", "method", "payload", "status", "updated_by",
	"retry_count", "retry_delay_ms", "timeout_seconds", "headers", "success_status_codes",
	"notify_fail_threshold", "notify_telegram", "notify_slack",
}

func (s *TaskService) UpdateTask(id string, input models.UpdateTaskReq) (*models.Task, error) {
	task := taskFromReq(id, input)

	// status 省略（空字串）時不更新該欄、沿用 DB 現值；否則 Select 會把 "" 寫進去，接著 SyncTask 取消排程
	cols := taskUpdatableColumns
	if input.Status == "" {
		cols = slices.DeleteFunc(slices.Clone(cols), func(c string) bool { return c == "status" })
	}
	result := s.db.Model(&models.Task{}).Where("id = ?", id).Select(cols).Updates(task)
	if result.Error != nil {
		return nil, fmt.Errorf("task: %T, db.Updates err: %v", task, result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	if err := scheduler.SyncTask(s.cronScheduler, id); err != nil {
		logger.Log.Errorf("task id: %s, SyncTask err: %v", id, err)
		return nil, err
	}
	// 重讀 DB：task 是請求映射出來的，status／created_at／group 不準
	return s.GetTask(id)
}

// UpdateTaskStatus 只切換任務啟用/停用狀態並對應 register/unregister cron job，
// 不需要像 UpdateTask 一樣送整包任務資料
func (s *TaskService) UpdateTaskStatus(id, status, userId string) (*models.Task, error) {
	var task models.Task
	if err := s.db.First(&task, "id = ?", id).Error; err != nil {
		return nil, err
	}

	result := s.db.Model(&models.Task{}).Where("id = ?", id).
		Updates(map[string]interface{}{"status": status, "updated_by": userId})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	task.Status = status
	task.UpdatedBy = userId

	if err := scheduler.SyncTask(s.cronScheduler, id); err != nil {
		logger.Log.Errorf("task id: %s, SyncTask err: %v", id, err)
		return nil, err
	}
	return &task, nil
}

// RunTaskNow 立即執行任務 repeat 次（間隔 intervalMs 毫秒），不受排程狀態影響，非同步執行後立刻回傳該批次的 batch id
func (s *TaskService) RunTaskNow(id string, repeat, intervalMs int) (string, error) {
	// 不走 GetTask：執行只需要任務本身，不需要 Preload("Group") 與 NextRunAt 裝飾
	var task models.Task
	if err := s.db.First(&task, "id = ?", id).Error; err != nil {
		return "", err
	}
	return scheduler.RunTaskNow(s.cronScheduler, &task, repeat, intervalMs)
}

// GetRunBatch 查詢「立即執行」批次的進度：running 表示批次是否仍在跑，logs 為該批次目前已寫入的紀錄（依開始時間升冪）。
// 順序不能反：必須先判斷 running 再查 logs。批次 goroutine 在所有紀錄都同步寫入完成後才會從排程器的 registry 消失，
// 所以先讀到 running=false 之後再查，logs 一定完整；若反過來先查 logs 再問 running，
// 可能在兩者之間又寫入了最後幾筆，造成「已完成但 logs 缺漏」。
func (s *TaskService) GetRunBatch(taskID, batchID string) (running bool, logs []models.TaskExecutionLog, err error) {
	running = scheduler.IsBatchRunning(s.cronScheduler, taskID, batchID)
	logs = []models.TaskExecutionLog{}
	err = s.db.Where("task_id = ? AND batch_id = ?", taskID, batchID).
		Order("started_at ASC, id ASC").Find(&logs).Error
	return running, logs, err
}

// CancelRunNow 取消任務目前進行中的「立即執行」重複批次
func (s *TaskService) CancelRunNow(id string) error {
	return scheduler.CancelTaskRun(s.cronScheduler, id)
}

func (s *TaskService) DeleteTask(id string) error {
	result := s.db.Where("id = ?", id).Delete(&models.Task{})
	if result.Error != nil {
		logger.Log.Errorf("task id: %s, db.Delete err: %v", id, result.Error)
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	s.syncTasks([]string{id})

	return nil
}

// ListTaskGroups 支援篩選（name 模糊比對、status）、排序與分頁
func (s *TaskService) ListTaskGroups(status, name, sortField, sortOrder string, limit, offset int) ([]models.TaskGroup, int64, error) {
	var groups []models.TaskGroup
	query := s.db.Model(&models.TaskGroup{})

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if name != "" {
		query = query.Where("name ILIKE ?", "%"+name+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if order := orderBy(listTaskGroupsSortableFields, sortField, sortOrder, ""); order != "" {
		query = query.Order(order)
	}

	err := query.Limit(limit).Offset(offset).Find(&groups).Error
	return groups, total, err
}

// GetTaskGroup 取得單一任務群組
func (s *TaskService) GetTaskGroup(id string) (*models.TaskGroup, error) {
	var group models.TaskGroup
	err := s.db.First(&group, "id = ?", id).Error
	return &group, err
}

func (s *TaskService) CreateTaskGroup(input models.CreateTaskGroupReq) (*models.TaskGroup, error) {
	group := &models.TaskGroup{
		ID:          xid.New().String(),
		Name:        input.Name,
		Description: input.Description,
		Status:      input.Status,
		CreatedBy:   input.UserId,
		UpdatedBy:   input.UserId,
	}

	if err := s.db.Create(group).Error; err != nil {
		return nil, err
	}

	return group, nil
}

// deactivateGroupTasksTx 在交易內把群組底下仍為 enabled 的任務批次轉為 disabled，
// 回傳被停用的任務清單（供呼叫端在交易 commit 後取消其 cron 排程）。
// 用單一 UPDATE ... RETURNING 而不是先 Find 再 Update：兩條敘述之間若有任務被別的請求改成
// enabled，Find-then-Update 會漏抓到它（UPDATE 仍會把它一併停用，但不會出現在回傳清單裡，
// 就永遠不會被 SyncTask 同步）。RETURNING 讓「哪些列被改」跟「改成什麼」在同一個敘述完成。
func (s *TaskService) deactivateGroupTasksTx(tx *gorm.DB, groupID string) ([]models.Task, error) {
	var tasks []models.Task
	err := tx.Model(&tasks).
		Clauses(clause.Returning{Columns: []clause.Column{{Name: "id"}, {Name: "name"}}}).
		Where("group_id = ? AND status = ?", groupID, models.STATUS_ENABLED).
		Update("status", models.STATUS_DISABLED).Error
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

// syncTasks 在 DB commit 之後一次同步一批任務的 cron 排程；失敗只記 log（DB 已是正確狀態，poller 會再補）
func (s *TaskService) syncTasks(ids []string) {
	if err := scheduler.SyncTasks(s.cronScheduler, ids); err != nil {
		logger.Log.Errorf("task ids: %v, SyncTasks err: %v", ids, err)
	}
}

func (s *TaskService) UpdateTaskGroup(id string, input models.UpdateTaskGroupReq) (*models.TaskGroup, error) {
	group := &models.TaskGroup{
		ID:          id,
		Name:        input.Name,
		Description: input.Description,
		Status:      input.Status,
		UpdatedBy:   input.UserId,
	}

	var deactivated []models.Task
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// status 省略（空字串）時不更新該欄、沿用 DB 現值，理由同 UpdateTask
		cols := []string{"name", "description", "updated_by"}
		if input.Status != "" {
			cols = append(cols, "status")
		}
		result := tx.Model(&models.TaskGroup{}).Where("id = ?", id).
			Select(cols).
			Updates(group)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		// 群組停用時連動停用底下仍為 active 的任務；改回啟用不自動恢復子任務
		if input.Status == models.STATUS_DISABLED {
			tasks, err := s.deactivateGroupTasksTx(tx, id)
			if err != nil {
				return err
			}
			deactivated = tasks
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("task group: %T, update err: %w", group, err)
	}

	ids := make([]string, len(deactivated))
	for i, t := range deactivated {
		ids[i] = t.ID
	}
	s.syncTasks(ids)

	// 重讀 DB，回應才反映沿用的 status／created_at 等現值
	return s.GetTaskGroup(id)
}

// DeleteTaskGroup 刪除群組前連動停用底下仍為 active 的任務並取消其排程，
// 回傳被停用的任務清單供 API 回應提示使用者
func (s *TaskService) DeleteTaskGroup(id string) ([]models.Task, error) {
	var deactivated []models.Task
	err := s.db.Transaction(func(tx *gorm.DB) error {
		tasks, err := s.deactivateGroupTasksTx(tx, id)
		if err != nil {
			return err
		}
		deactivated = tasks

		result := tx.Where("id = ?", id).Delete(&models.TaskGroup{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("task group id: %s, delete err: %w", id, err)
	}

	ids := make([]string, len(deactivated))
	for i, t := range deactivated {
		ids[i] = t.ID
	}
	s.syncTasks(ids)

	return deactivated, nil
}
