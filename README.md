專案資料夾結構
```plaintext
scheduler-tool/
├── cmd/
│   └── migrate/            # 資料庫遷移工具 (go run cmd/migrate/migrate.go)
│       └── migrate.go
├── config/                 # 配置檔案和初始化
│   └── config.go
├── docker/                 # 本地開發用 docker-compose（mysql、redis，redis 目前未被程式使用）
│   ├── docker-compose.yaml
│   ├── mysql/
│   └── mysql_init.sql
├── internal/               # 私有應用程式邏輯
│   ├── app/                # 依賴組裝 (AppServices)
│   │   └── app_services.go
│   ├── auth/               # JWT 簽發/解析
│   │   └── jwt.go
│   ├── database/           # 資料庫連線和初始化
│   │   └── database.go
│   ├── controllers/        # 控制器層（處理 API 請求）
│   │   ├── auth_controller.go
│   │   └── task_controller.go # 任務與群組管理控制器
│   ├── models/             # 資料結構和資料庫模型
│   │   ├── user.go
│   │   └── task.go         # 任務、群組模型
│   ├── services/           # 業務邏輯層，直接操作 *gorm.DB（無獨立的 repositories 層）
│   │   ├── auth_service.go
│   │   └── task_service.go # 任務與群組服務
│   ├── scheduler/          # cron 排程器，執行 http/rpc 任務
│   │   ├── cron.go
│   │   └── jobs.go
│   ├── middleware/         # 中介軟體
│   │   └── auth_middleware.go
│   └── router/             # 路由設定
│       └── router.go
├── pkg/                    # 公用元件（工具、輔助功能等）
│   ├── logger/             # 日誌記錄
│   ├── response/           # API 回應封裝
│   └── utils/              # 驗證器等公用工具
├── .env                    # 環境變數配置
├── go.mod                  # Go modules 配置
└── README.md               # 專案說明文件

```