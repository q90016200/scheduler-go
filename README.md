專案資料夾結構
```plaintext
scheduler-tool/
├── cmd/
│   └── migrate/            # 資料庫遷移工具 (go run cmd/migrate/migrate.go)
│       └── migrate.go
├── config/                 # 配置檔案和初始化
│   └── config.go
├── docker/                 # 本地開發用 docker-compose（postgres）
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
│   │   ├── task_execution_log_controller.go # 任務執行紀錄與（全站/單一任務）儀表板控制器
│   │   └── notify_controller.go         # 回報通知管道是否已在 .env 設定（不回傳憑證本身）
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
│   ├── notify/             # 失敗通知管道（telegram.go / slack.go / email.go），由 jobs.go 呼叫
│   ├── ssrf/               # 任務執行連線的 SSRF 防護（ssrf.go，見下方「安全性」）
│   ├── middleware/         # 中介軟體
│   │   ├── auth_middleware.go
│   │   └── tracing_middleware.go # trace_id（沿用 X-Request-ID 或自動產生）
│   └── router/             # 路由設定，同時負責服務 web/dist 前端靜態檔（SPA fallback）
│       └── router.go
├── pkg/                    # 公用元件（工具、輔助功能等）
│   ├── logger/             # 日誌記錄（stdout 彩色 / 檔案 JSON 分流，見下方「日誌與追蹤」）
│   ├── response/           # API 回應封裝
│   └── utils/              # 驗證器（validator.go）等公用工具
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

   會啟動 `postgres:16`（5432，資料存放於 `docker/postgres/data`）。要停止則執行 `docker compose down`。

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

   涵蓋 `internal/scheduler`（cron 註冊與排程、HTTP 成功/重試判定、立即執行）、`internal/ssrf`（SSRF 連線防護）與 `pkg/utils`（驗證器）。排程相關程式碼可另外用 `go test -race ./internal/scheduler/...` 檢查併發安全。

## Docker 部署

repo 根目錄的 `Dockerfile` 為多階段建置（前端 `npm run build` → Go 編譯 `server` 與 `migrate` → `alpine` 執行映像，非 root 使用者）。設定一律走環境變數，映像內不含 `.env`：

```bash
docker build -t schedule-go .
docker run --rm --env-file .env schedule-go ./migrate   # 遷移
docker run -d -p 8080:8080 --env-file .env schedule-go   # 啟動
```

`.env` 裡的 `POSTGRES_HOST` 要改成容器可達的位址（例如 `host.docker.internal`，而非 `127.0.0.1`）。映像內建 `HEALTHCHECK`，打 `GET /healthz`。

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
| `TIMEZONE` | 應用時區（IANA 名稱）。cron 排程、資料庫連線 `TimeZone`、日誌時間戳（`time.Local`）都以此為準，不受主機/容器本機時區影響；無效值會在啟動時 `log.Fatalf`。單一任務可在排程前加 `CRON_TZ=` 覆寫，如 `CRON_TZ=Asia/Tokyo 0 9 * * *` | `Asia/Taipei` |
| `LOG_LEVEL` | logrus 最低輸出等級（`debug`/`info`/`warn`/`error`...） | 開發環境 `debug`，其餘 `info` |
| `POSTGRES_*` | PostgreSQL 連線資訊。任一項缺少同樣會在啟動時 `log.Fatalf` | - |
| `TELEGRAM_BOT_TOKEN` / `TELEGRAM_CHAT_ID` | 失敗通知的 Telegram 管道；任一項留空視為停用 | - |
| `SLACK_WEBHOOK_URL` | 失敗通知的 Slack 管道（Incoming Webhook URL）；留空視為停用 | - |
| `SMTP_HOST` / `SMTP_PORT` / `SMTP_USER` / `SMTP_PASSWORD` / `SMTP_FROM` / `SMTP_TO` | 失敗通知的 Email 管道；`SMTP_HOST`/`SMTP_FROM`/`SMTP_TO` 任一項留空視為停用 | - |

## API 端點

> **登入/註冊目前暫時停用**：`JWTAuthMiddleware` 未掛載在任何路由上，前端也拿掉了未登入導向 `/login` 的守衛，所有端點與頁面皆可直接使用，不需登入。以下「需登入」標註為停用前的原始設計，之後要恢復可參考 `internal/router/router.go`。

**錯誤回應格式**：所有回應皆為 `{code, message, data, error, trace}`。驗證失敗（400）時 `error` 為 `[{field, tag, param}]` 陣列，`field` 使用 JSON 欄位名（如 `endpoint`、`retry_count`），前端據此翻成中文欄位與錯誤訊息（`web/src/api/client.ts`）。

**健康檢查**：`GET /healthz`（在 `/api` 之外，不掛 tracing，也不寫進 gin access log）。資料庫 ping 成功回 200 `{"status":"ok"}`，失敗回 503，供容器/負載平衡器探測。

**Auth**（`/api/auth`）：`POST /register`、`POST /login`（設定 `auth_token` HttpOnly cookie；帳號 `status` 為 `disabled` 時登入會被拒絕）、`POST /logout`、`GET /me`（取得目前使用者，需帶有效 cookie 才有意義）。

**User**（`/api/users`）：
- `GET /`（清單，可用 `name`（模糊比對）/`limit`/`offset`）、`GET /:id`、`POST /`、`PUT /:id`（`password` 留空代表不更改密碼）、`DELETE /:id`
- `PATCH /:id/status`（body 僅需 `{"status": "enabled"|"disabled"}`，只切換啟用狀態）
- 密碼一律以 bcrypt 雜湊儲存；`status` 沿用 Task/TaskGroup 共用的 `enabled`/`disabled` 慣例。

**Task**（`/api/tasks`）：
- `GET /`（清單，可用 `status`/`group_id`/`name`（模糊比對）/`sort_field`/`sort_order`/`limit`/`offset` 查詢參數）；每筆任務會多帶 `next_run_at`（unix 秒，只有已啟用且成功掛上 cron job 的任務才有值，其餘為 `null`）
- `GET /:id`、`POST /`、`PUT /:id`（整包更新，`name`/`schedule`/`type`/`endpoint` 皆為必填）、`DELETE /:id`
- `PATCH /:id/status`（body 僅需 `{"status": "enabled"|"disabled", "user_id": "..."}`，只切換啟用狀態，不需送整包任務資料）
- `POST /:id/run`（立即執行，不受排程狀態影響；非同步觸發，呼叫後立刻回傳）。body 可選帶 `{"repeat": 1-50, "interval_ms": 100-60000}` 指定重複執行次數與每次呼叫的間隔毫秒數，省略時預設執行 1 次；同一任務同時只允許一個進行中的重複批次，重複觸發會回 409。中途某次執行失敗不會中斷剩下的次數，每次執行都各自寫入一筆執行紀錄。立即執行（手動觸發）不套用失敗通知門檻計數，避免測試時誤觸發通知。
- `DELETE /:id/run`（取消該任務目前進行中的重複批次；會立刻中斷正在進行中的那一發 HTTP/RPC 呼叫與剩餘的重試，不必等它自然逾時或重試跑完，中斷的那筆會記為失敗、錯誤訊息帶 `context canceled`；找不到進行中的批次回 404）
- `GET /:id/stats`（單一任務執行儀表板小面板：近 30 天成功率 `success_rate_30d`、近 30 天平均耗時 `avg_duration_ms_30d`（毫秒）、最近 10 次中失敗次數 `recent_fail_count`；只採計排程自動觸發、且是該次觸發最終結果的紀錄，即時 SQL 聚合，不做快取）
- 建立/更新時 `type` 限定 `http`/`rpc`、`schedule` 必須是合法的標準 cron 表達式，兩者驗證失敗會在寫入資料庫前被擋下（回 400），不會留下無法排程的孤兒資料。
- 任務新增以下欄位（皆選填，有預設值）：
  - `retry_count`（0-5，失敗時額外重試次數）、`retry_delay_ms`（0-30000，每次重試間隔毫秒，固定值不做指數退避）
  - `timeout_seconds`（1-120，單次 HTTP/RPC 呼叫逾時秒數，預設 10）
  - `headers`（自由 JSON 字串，如 `{"Authorization":"Bearer xxx"}`；http 任務轉成 request headers、rpc 任務轉成 gRPC metadata；明文儲存，跟 `payload`/`endpoint` 同信任模型）
  - `success_status_codes`（僅 http 任務；逗號分隔的狀態碼或類別，如 `200,201,2xx`，最多 100 字元。留空沿用預設：HTTP 400 以下視為成功。有設定時，符合者即成功（可把 `404` 之類設為成功）；不符合者失敗，其中 5xx 仍可重試、其餘不重試）
  - `notify_fail_threshold`（1-20，連續失敗達此次數才發一次失敗通知，之後靜音直到成功一次重新累積；0 表示不啟用通知）
  - `notify_telegram`/`notify_slack`/`notify_email`（布林，個別勾選要不要收該管道的通知；管道本身要在 `.env` 設定好才會真的送出）

**TaskGroup**（`/api/tasks/group`）：`GET /`（可用 `status`/`name`/`sort_field`/`sort_order`/`limit`/`offset`）、`GET /:id`、`POST /`、`PUT /:id`、`DELETE /:id`。將群組 `PUT` 成 `disabled`，或直接 `DELETE` 群組，都會連動把該群組底下仍為 `enabled` 的任務改成 `disabled` 並取消排程（`DELETE` 的回應會帶 `deactivated_tasks: [{id, name}]`，供前端提示使用者哪些任務被停用了）；改回 `enabled` 不會自動恢復子任務，需個別開啟。刪除群組不會刪除底下的任務，只會解除關聯（任務的 `group_id` 變成 `null`）——前端刪除前的確認對話框會列出底下所有任務（含已停用）的總數與名稱，避免使用者不知道這件事。

**執行紀錄**（`/api/tasks/logs`）：`GET /`（可用 `task_id`/`task_name`/`group_id` 篩選，`limit`/`offset` 分頁，回傳每筆會帶 `task.name`/`task.group.name`）、`GET /:id`（同樣帶 `task` 資訊）。刪除任務會連動刪除其執行紀錄（外鍵 `ON DELETE CASCADE`）。每筆紀錄多帶：
- `source`（`cron`/`manual`，區分是排程自動觸發還是立即執行手動觸發）
- `attempt_number`（第幾次嘗試，第一次是 1）、`is_final_attempt`（是否為這次觸發（含所有重試）的最後一筆；設有 `retry_count` 的任務失敗重試時，每次嘗試各存一筆，只有最後一筆會是 `true`）

**失敗重試與通知**：任務失敗（網路錯誤/逾時/HTTP 5xx）時依 `retry_count`/`retry_delay_ms` 重試；HTTP 4xx 與參數格式錯誤（例如 method/payload 格式不對）視為不可重試，直接判定為這次觸發的最終結果。只有排程自動觸發（`source=cron`）的最終結果才會累加/清零任務的連續失敗次數：失敗 +1、成功清零；連續失敗次數剛好等於 `notify_fail_threshold` 的那一刻，依任務勾選的管道（Telegram/Slack/Email）非同步發送一次通知，之後即使繼續失敗也靜音，直到成功一次重新累積才會再通知一次。通知發送失敗只記 log，不重試、不影響任務執行本身。

**儀表板**（`/api/dashboard`）：`GET /stats`，回傳全站統計——啟用/停用任務數、近 24 小時執行次數與成功率、近 24 小時失敗次數 Top 5 任務、目前連續失敗中的任務清單；統計口徑與 `/tasks/:id/stats` 相同（只採計 `source=cron` 且為該次觸發最終結果的紀錄），即時 SQL 聚合，不做快取。

**通知設定狀態**（`/api/notify`）：`GET /config`，回傳 `{telegram_configured, slack_configured, email_configured}` 三個布林值，只看對應的 `.env` 欄位是否已填，不回傳任何 token/密碼本身；供前端在任務表單提醒使用者「勾了某管道但 `.env` 沒設定，通知不會送出」。

## 安全性

- **SSRF 防護**：`http`/`rpc` 任務執行時，連線會擋下 loopback（`127.0.0.1`）、link-local（含雲端 metadata 位址 `169.254.169.254`）與未指定位址（`0.0.0.0`），且是在實際連線當下檢查已解析出的 IP，不受 DNS rebinding 影響。**刻意不封鎖 RFC1918 私有網段**——這是內部排程工具，打內網服務是其本職功能。確實需要打 loopback 時可設定 `SSRF_ALLOW_PRIVATE_TARGETS=true`，但這會關閉整組防護，請謹慎使用。
- **登入/註冊目前暫時停用**（見上方端點列表前的說明），所有端點無需認證即可呼叫；請只在受信任的內部網路中部署。

## 日誌與追蹤

- `logs/app.log`（`lumberjack` 輪替：10MB/份、保留 5 份、10 天、壓縮）以 **JSON** 格式寫入，可直接用 `jq` 過濾，例如：`jq 'select(.trace_id=="xxxxx")' logs/app.log`。
- 終端機（stdout）輸出彩色、人類可讀的格式，兩者格式不同但內容一致（含 `trace_id`、錯誤內容等欄位）。
- 每個 API 請求都會被指派一個 `trace_id`：若請求帶有 `X-Request-ID` header 就沿用，否則自動產生；同一個值會出現在 API 回應的 `trace` 欄位、回應的 `X-Request-ID` header，以及該請求相關的所有 log 行中，方便串接排查。
- 輸出等級由 `LOG_LEVEL` 環境變數控制（見上方環境變數表）。
- 5xx 錯誤後端不回傳內部錯誤細節（只寫進日誌），前端的錯誤提示會附上 `（trace: xxx）`，回報問題時提供此值即可用上面的 `jq` 指令查到對應日誌。

## 前端開發

```bash
cd web
npm install
npm run dev      # 開發伺服器，透過 vite.config.ts 的 proxy 打後端 /api（後端埠設定於 web/.env.development 的 VITE_BACKEND_PORT）
npm run build    # 產出 web/dist，供 Go 後端同源服務
```

正式部署：`cd web && npm run build` 產出 `web/dist/`，與 Go 執行檔放在同一相對路徑下啟動（`go run main.go` 或編譯後的 binary），`internal/router/router.go` 會直接服務這個資料夾並提供 SPA fallback，不需要額外的 nginx。開發模式下 Vite dev server 也透過 proxy 讓瀏覽器端請求保持同源，因此整個專案不需要 CORS middleware。

首頁為 `/dashboard`（全站儀表板），任務清單頁每 120 秒、其餘清單頁與詳情頁每 30 秒自動背景刷新（不會蓋整頁 loading，靜默換資料），可在導覽列右上角的開關關閉，設定存在瀏覽器 `localStorage`、全站共用。