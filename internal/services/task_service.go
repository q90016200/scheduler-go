package services

import (
	"fmt"
	"github.com/q90016200/scheduler-go/internal/models"
	"github.com/q90016200/scheduler-go/internal/scheduler"
	"github.com/q90016200/scheduler-go/pkg/logger"
	"github.com/rs/xid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	for i := range tasks {
		decorateNextRunAt(s.cronScheduler, &tasks[i])
	}
	return tasks, total, err
}

// GetTask 取得單一任務
func (s *TaskService) GetTask(id string) (*models.Task, error) {
	var task models.Task
	if err := s.db.Preload("Group").First(&task, "id = ?", id).Error; err != nil {
		return nil, err
	}
	decorateNextRunAt(s.cronScheduler, &task)
	return &task, nil
}

// decorateNextRunAt 從記憶體中的 cron entry 查下次執行時間裝飾進 Task；
// 未掛上 cron job（未啟用、或註冊失敗）的任務保持 nil，前端顯示「-」
func decorateNextRunAt(cs *scheduler.CronScheduler, task *models.Task) {
	next, ok := scheduler.NextRunAt(cs, task.ID)
	if !ok {
		return
	}
	unix := next.Unix()
	task.NextRunAt = &unix
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
		task.TimeoutSeconds = 10
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

	// 當是執行的放入 syncMap 內，供排程取得目前可執行 task 清單
	if task.Status == models.STATUS_ENABLED {
		if err := scheduler.RegisterTaskJob(s.cronScheduler, task); err != nil {
			// 註冊排程失敗，不留下無法排程的孤兒資料
			if delErr := s.db.Delete(&models.Task{}, "id = ?", task.ID).Error; delErr != nil {
				logger.Log.Errorf("task id: %s, 註冊排程失敗後清理孤兒資料也失敗: %v", task.ID, delErr)
			}
			return nil, err
		}
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

	result := s.db.Model(&models.Task{}).Where("id = ?", id).Select(taskUpdatableColumns).Updates(task)
	if result.Error != nil {
		return nil, fmt.Errorf("task: %T, db.Updates err: %v", task, result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	// 重新啟動排程任務
	scheduler.UnregisterTaskJob(s.cronScheduler, id)
	if task.Status == models.STATUS_ENABLED {
		if err := scheduler.RegisterTaskJob(s.cronScheduler, task); err != nil {
			errMsg := fmt.Sprintf("task: %T, RegisterTaskJob err: %v", task, err)
			logger.Log.Error(errMsg)
			return nil, err
		}
	}

	return task, nil
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

	scheduler.UnregisterTaskJob(s.cronScheduler, id)
	if status == models.STATUS_ENABLED {
		if err := scheduler.RegisterTaskJob(s.cronScheduler, &task); err != nil {
			errMsg := fmt.Sprintf("task: %T, RegisterTaskJob err: %v", task, err)
			logger.Log.Error(errMsg)
			return nil, err
		}
	}

	return &task, nil
}

// RunTaskNow 立即執行任務 repeat 次（間隔 intervalMs 毫秒），不受排程狀態影響，非同步執行後立刻回傳
func (s *TaskService) RunTaskNow(id string, repeat, intervalMs int) (*models.Task, error) {
	task, err := s.GetTask(id)
	if err != nil {
		return nil, err
	}
	if err := scheduler.RunTaskNow(s.cronScheduler, task, repeat, intervalMs); err != nil {
		return nil, err
	}
	return task, nil
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

	scheduler.UnregisterTaskJob(s.cronScheduler, id)

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
// 就永遠不會呼叫 UnregisterTaskJob）。RETURNING 讓「哪些列被改」跟「改成什麼」在同一個敘述完成。
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

// unregisterTasks 取消一批任務的 cron 排程（純記憶體操作，於交易 commit 後呼叫）
func (s *TaskService) unregisterTasks(tasks []models.Task) {
	for _, task := range tasks {
		scheduler.UnregisterTaskJob(s.cronScheduler, task.ID)
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
		result := tx.Model(&models.TaskGroup{}).Where("id = ?", id).
			Select("name", "description", "status", "updated_by").
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

	s.unregisterTasks(deactivated)

	return group, nil
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

	s.unregisterTasks(deactivated)

	return deactivated, nil
}
