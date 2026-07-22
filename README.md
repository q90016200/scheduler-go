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
│   │   ├── task_controller.go           # 任務與群組管理控制器
│   │   └── task_execution_log_controller.go # 任務執行紀錄控制器
│   ├── models/             # 資料結構和資料庫模型
│   │   ├── user.go
│   │   ├── task.go                # 任務、群組模型
│   │   └── task_execution_log.go  # 任務執行紀錄模型
│   ├── services/           # 業務邏輯層，直接操作 *gorm.DB（無獨立的 repositories 層）
│   │   ├── auth_service.go
│   │   ├── task_service.go            # 任務與群組服務
│   │   └── task_execution_log_service.go # 任務執行紀錄服務
│   ├── scheduler/          # cron 排程器，執行 http/rpc 任務、記錄執行結果、清除過期紀錄
│   │   ├── cron.go
│   │   └── jobs.go
│   ├── middleware/         # 中介軟體
│   │   ├── auth_middleware.go
│   │   └── cors_middleware.go   # CORS（僅開發模式掛載）
│   └── router/             # 路由設定，同時負責服務 web/dist 前端靜態檔（SPA fallback）
│       └── router.go
├── pkg/                    # 公用元件（工具、輔助功能等）
│   ├── logger/             # 日誌記錄
│   ├── response/           # API 回應封裝
│   └── utils/              # 驗證器等公用工具
├── web/                    # Vue3 + TypeScript + Vite 前端（獨立專案，見下方「前端開發」）
├── .env                    # 環境變數配置
├── go.mod                  # Go modules 配置
└── README.md               # 專案說明文件

```

## 環境變數 (.env)

| 變數 | 說明 | 預設 |
|---|---|---|
| `APP_ENV` | `development` 時才會掛載 CORS middleware（正式環境同源部署不需要） | `development` |
| `SERVER_PORT` | Gin 監聽埠 | - |
| `SERVER_DOMAIN` | cookie 的 domain | - |
| `APP_SECRET` | JWT 簽章金鑰 | - |
| `TOKEN_EXPIRY` | JWT/cookie 有效秒數 | - |
| `CORS_ALLOW_ORIGINS` | 開發模式允許的前端來源，逗號分隔 | 空 |
| `EXECUTION_LOG_RETENTION_DAYS` | 任務執行紀錄保留天數，超過會被每日排程清除 | `120` |
| `POSTGRES_*` | PostgreSQL 連線資訊 | - |

## API 端點

> **登入/註冊目前暫時停用**：`JWTAuthMiddleware` 未掛載在任何路由上，前端也拿掉了未登入導向 `/login` 的守衛，所有端點與頁面皆可直接使用，不需登入。以下「需登入」標註為停用前的原始設計，之後要恢復可參考 `internal/router/router.go`。

**Auth**（`/api/auth`）：`POST /register`、`POST /login`（設定 `auth_token` HttpOnly cookie）、`POST /logout`、`GET /me`（取得目前使用者，需帶有效 cookie 才有意義）。

**Task**（`/api/tasks`）：`GET /`（清單，可用 `status`/`sort_field`/`sort_order` 查詢參數）、`GET /:id`、`POST /`、`PUT /:id`、`DELETE /:id`。

**TaskGroup**（`/api/tasks/group`）：`GET /`、`GET /:id`、`POST /`、`PUT /:id`、`DELETE /:id`。

**執行紀錄**（`/api/tasks/logs`）：`GET /`（可用 `task_id` 篩選，`limit`/`offset` 分頁）、`GET /:id`。

## 前端開發

```bash
cd web
npm install
npm run dev      # 開發伺服器，透過 vite.config.ts 的 proxy 打後端 /api（後端埠設定於 web/.env.development 的 VITE_BACKEND_PORT）
npm run build    # 產出 web/dist，供 Go 後端同源服務
```

正式部署：`cd web && npm run build` 產出 `web/dist/`，與 Go 執行檔放在同一相對路徑下啟動（`go run main.go` 或編譯後的 binary），`internal/router/router.go` 會直接服務這個資料夾並提供 SPA fallback，不需要額外的 nginx，也不需要開啟 CORS。