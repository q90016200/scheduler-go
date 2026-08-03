專案資料夾結構
```plaintext
scheduler-tool/
├── cmd/
│   └── migrate/            # 資料庫遷移工具 (go run cmd/migrate/migrate.go)
│       └── migrate.go
├── config/                 # 配置檔案和初始化
│   └── config.go
├── docker/                 # 本地開發用 docker-compose（postgres、redis，redis 目前未被程式使用）
│   ├── docker-compose.yaml
│   └── postgres/
├── internal/               # 私有應用程式邏輯
│   ├── app/                # 依賴組裝 (AppServices)
│   │   └── app_services.go
│   ├── auth/               # JWT 簽發/解析
│   │   └── jwt.go
│   ├── database/           # 資料庫連線和初始化
│   │   └── database.go
│   ├── controllers/        # 控制器層（處理 API 請求）
│   │   ├── auth_controller.go
│   │   ├── user_controller.go           # 使用者管理控制器
│   │   ├── task_controller.go           # 任務與群組管理控制器
│   │   └── task_execution_log_controller.go # 任務執行紀錄控制器
│   ├── models/             # 資料結構和資料庫模型
│   │   ├── user.go
│   │   ├── task.go                # 任務、群組模型
│   │   └── task_execution_log.go  # 任務執行紀錄模型
│   ├── services/           # 業務邏輯層，直接操作 *gorm.DB（無獨立的 repositories 層）
│   │   ├── auth_service.go
│   │   ├── user_service.go            # 使用者管理服務
│   │   ├── task_service.go            # 任務與群組服務
│   │   └── task_execution_log_service.go # 任務執行紀錄服務
│   ├── scheduler/          # cron 排程器，執行 http/rpc 任務、記錄執行結果、清除過期紀錄
│   │   ├── cron.go              # cron.Recover 防護、每分鐘 poller、SSRF 防護的 resty client
│   │   └── jobs.go
│   ├── middleware/         # 中介軟體
│   │   ├── auth_middleware.go
│   │   └── tracing_middleware.go # trace_id（沿用 X-Request-ID 或自動產生）
│   └── router/             # 路由設定，同時負責服務 web/dist 前端靜態檔（SPA fallback）
│       └── router.go
├── pkg/                    # 公用元件（工具、輔助功能等）
│   ├── logger/             # 日誌記錄（stdout 彩色 / 檔案 JSON 分流，見下方「日誌與追蹤」）
│   ├── response/           # API 回應封裝
│   └── utils/              # 驗證器、SSRF 連線防護（ssrf.go）等公用工具
├── web/                    # Vue3 + TypeScript + Vite 前端（獨立專案，見下方「前端開發」）
├── .env                    # 環境變數配置
├── go.mod                  # Go modules 配置
└── README.md               # 專案說明文件

```

## 快速開始

1. **設定環境變數**

   ```bash
   cp .env.example .env
   ```

   預設值已對應下一步 docker-compose 起的 postgres（`schedule_go` / `postgres` / `postgres` / 5432），本機開發可直接使用不需修改。

2. **啟動資料庫**

   ```bash
   cd docker
   docker compose up -d
   ```

   會啟動 `postgres:16`（5432，資料存放於 `docker/postgres/data`）與 `redis`（6379，目前程式尚未使用，可自行從 `docker-compose.yaml` 註解掉）。要停止則執行 `docker compose down`。

3. **執行資料庫遷移**

   回到 repo 根目錄執行（`migrate.go` 讀取的是根目錄的 `.env`，必須在此路徑下跑）：

   ```bash
   go run cmd/migrate/migrate.go
   ```

   會自動建立/更新 `User`、`Task`、`TaskGroup`、`TaskExecutionLog` 四張表，schema 有異動時重跑即可。

4. **啟動後端**

   ```bash
   go run main.go
   ```

   監聽 `SERVER_PORT`（預設 `8080`）。啟動後 cron 排程器會每分鐘掃描 `status = enabled` 的任務並執行。

5. **啟動前端**

   參考下方「前端開發」段落，`cd web && npm install && npm run dev`，開發伺服器預設在 http://localhost:5173。

6. **執行測試**（選用）

   ```bash
   go test ./...
   ```

   涵蓋 `internal/scheduler`（cron 註冊/錯誤路徑，以及立即執行的重複次數、併發批次拒絕、取消）與 `pkg/utils`（SSRF 連線防護）。排程相關程式碼可另外用 `go test -race ./internal/scheduler/...` 檢查併發安全。

## 環境變數 (.env)

| 變數 | 說明 | 預設 |
|---|---|---|
| `GIN_MODE` | Gin 執行模式（`debug`/`release`） | `debug` |
| `APP_ENV` | 應用程式環境（`development`/production 等），影響預設 `LOG_LEVEL` | `development` |
| `SERVER_PORT` | Gin 監聽埠 | - |
| `SERVER_DOMAIN` | cookie 的 domain | - |
| `APP_SECRET` | JWT 簽章金鑰。缺少會在啟動時 `log.Fatalf`，不會帶著空金鑰跑起來 | - |
| `TOKEN_EXPIRY` | JWT/cookie 有效秒數 | - |
| `EXECUTION_LOG_RETENTION_DAYS` | 任務執行紀錄保留天數，超過會被每日排程清除 | `120` |
| `SSRF_ALLOW_PRIVATE_TARGETS` | 設 `true` 時關閉任務執行的 SSRF 防護（連 loopback/metadata 位址都放行），僅供確實需要打 loopback 的情境使用 | `false` |
| `LOG_LEVEL` | logrus 最低輸出等級（`debug`/`info`/`warn`/`error`...） | 開發環境 `debug`，其餘 `info` |
| `POSTGRES_*` | PostgreSQL 連線資訊。任一項缺少同樣會在啟動時 `log.Fatalf` | - |

## API 端點

> **登入/註冊目前暫時停用**：`JWTAuthMiddleware` 未掛載在任何路由上，前端也拿掉了未登入導向 `/login` 的守衛，所有端點與頁面皆可直接使用，不需登入。以下「需登入」標註為停用前的原始設計，之後要恢復可參考 `internal/router/router.go`。

**Auth**（`/api/auth`）：`POST /register`、`POST /login`（設定 `auth_token` HttpOnly cookie；帳號 `status` 為 `disabled` 時登入會被拒絕）、`POST /logout`、`GET /me`（取得目前使用者，需帶有效 cookie 才有意義）。

**User**（`/api/users`）：
- `GET /`（清單，可用 `name`（模糊比對）/`limit`/`offset`）、`GET /:id`、`POST /`、`PUT /:id`（`password` 留空代表不更改密碼）、`DELETE /:id`
- `PATCH /:id/status`（body 僅需 `{"status": "enabled"|"disabled"}`，只切換啟用狀態）
- 密碼一律以 bcrypt 雜湊儲存；`status` 沿用 Task/TaskGroup 共用的 `enabled`/`disabled` 慣例。

**Task**（`/api/tasks`）：
- `GET /`（清單，可用 `status`/`group_id`/`name`（模糊比對）/`sort_field`/`sort_order`/`limit`/`offset` 查詢參數）
- `GET /:id`、`POST /`、`PUT /:id`（整包更新，`name`/`schedule`/`type`/`endpoint` 皆為必填）、`DELETE /:id`
- `PATCH /:id/status`（body 僅需 `{"status": "enabled"|"disabled", "user_id": "..."}`，只切換啟用狀態，不需送整包任務資料）
- `POST /:id/run`（立即執行，不受排程狀態影響；非同步觸發，呼叫後立刻回傳）。body 可選帶 `{"repeat": 1-50, "interval_ms": 100-60000}` 指定重複執行次數與每次呼叫的間隔毫秒數，省略時預設執行 1 次；同一任務同時只允許一個進行中的重複批次，重複觸發會回 409。中途某次執行失敗不會中斷剩下的次數，每次執行都各自寫入一筆執行紀錄。
- `DELETE /:id/run`（取消該任務目前進行中的重複批次，只會阻止尚未發出的下一次執行，已送出的那一發仍會跑完；找不到進行中的批次回 404）
- 建立/更新時 `type` 限定 `http`/`rpc`、`schedule` 必須是合法的標準 cron 表達式，兩者驗證失敗會在寫入資料庫前被擋下（回 400），不會留下無法排程的孤兒資料。

**TaskGroup**（`/api/tasks/group`）：`GET /`（可用 `status`/`name`/`sort_field`/`sort_order`/`limit`/`offset`）、`GET /:id`、`POST /`、`PUT /:id`、`DELETE /:id`。將群組 `PUT` 成 `disabled`，或直接 `DELETE` 群組，都會連動把該群組底下仍為 `enabled` 的任務改成 `disabled` 並取消排程（`DELETE` 的回應會帶 `deactivated_tasks: [{id, name}]`，供前端提示使用者哪些任務被停用了）；改回 `enabled` 不會自動恢復子任務，需個別開啟。刪除群組不會刪除底下的任務，只會解除關聯（任務的 `group_id` 變成 `null`）——前端刪除前的確認對話框會列出底下所有任務（含已停用）的總數與名稱，避免使用者不知道這件事。

**執行紀錄**（`/api/tasks/logs`）：`GET /`（可用 `task_id`/`task_name`/`group_id` 篩選，`limit`/`offset` 分頁，回傳每筆會帶 `task.name`/`task.group.name`）、`GET /:id`（同樣帶 `task` 資訊）。刪除任務會連動刪除其執行紀錄（外鍵 `ON DELETE CASCADE`）。

## 安全性

- **SSRF 防護**：`http`/`rpc` 任務執行時，連線會擋下 loopback（`127.0.0.1`）、link-local（含雲端 metadata 位址 `169.254.169.254`）與未指定位址（`0.0.0.0`），且是在實際連線當下檢查已解析出的 IP，不受 DNS rebinding 影響。**刻意不封鎖 RFC1918 私有網段**——這是內部排程工具，打內網服務是其本職功能。確實需要打 loopback 時可設定 `SSRF_ALLOW_PRIVATE_TARGETS=true`，但這會關閉整組防護，請謹慎使用。
- **登入/註冊目前暫時停用**（見上方端點列表前的說明），所有端點無需認證即可呼叫；請只在受信任的內部網路中部署。

## 日誌與追蹤

- `logs/app.log`（`lumberjack` 輪替：10MB/份、保留 5 份、10 天、壓縮）以 **JSON** 格式寫入，可直接用 `jq` 過濾，例如：`jq 'select(.trace_id=="xxxxx")' logs/app.log`。
- 終端機（stdout）輸出彩色、人類可讀的格式，兩者格式不同但內容一致（含 `trace_id`、錯誤內容等欄位）。
- 每個 API 請求都會被指派一個 `trace_id`：若請求帶有 `X-Request-ID` header 就沿用，否則自動產生；同一個值會出現在 API 回應的 `trace` 欄位、回應的 `X-Request-ID` header，以及該請求相關的所有 log 行中，方便串接排查。
- 輸出等級由 `LOG_LEVEL` 環境變數控制（見上方環境變數表）。

## 前端開發

```bash
cd web
npm install
npm run dev      # 開發伺服器，透過 vite.config.ts 的 proxy 打後端 /api（後端埠設定於 web/.env.development 的 VITE_BACKEND_PORT）
npm run build    # 產出 web/dist，供 Go 後端同源服務
```

正式部署：`cd web && npm run build` 產出 `web/dist/`，與 Go 執行檔放在同一相對路徑下啟動（`go run main.go` 或編譯後的 binary），`internal/router/router.go` 會直接服務這個資料夾並提供 SPA fallback，不需要額外的 nginx。開發模式下 Vite dev server 也透過 proxy 讓瀏覽器端請求保持同源，因此整個專案不需要 CORS middleware。