專案資料夾結構 (待調整)
```plaintext
scheduler-tool/
├── cmd/                  # 命令行工具入口，例如 migrate 或 main
│   └── migrate/          # 單獨的遷移工具
│       └── migrate.go
├── config/                 # 配置檔案和初始化
│   └── config.go           # 配置初始化
├── internal/               # 私有應用程式邏輯
│   ├── database/           # 資料庫連線和初始化
│   │   └── database.go
│   ├── controllers/        # 控制器層（處理 API 請求）
│   │   ├── auth.go         # 驗證控制器
│   │   ├── task.go         # 任務管理控制器
│   │   └── group.go        # 群組管理控制器
│   ├── models/             # 資料結構和資料庫模型
│   │   ├── user.go         # 用戶模型
│   │   ├── task.go         # 任務、群組模型
│   │   └── log.go          # 任務執行紀錄模型
│   ├── repositories/       # 資料存取層（與資料庫互動）
│   │   ├── user_repo.go    # 用戶資料庫操作
│   │   ├── task_repo.go    # 任務資料庫操作
│   │   └── group_repo.go   # 群組資料庫操作
│   ├── services/           # 業務邏輯層（封裝核心邏輯）
│   │   ├── auth_service.go # 驗證服務
│   │   ├── task_service.go # 任務服務
│   │   └── group_service.go# 群組服務
│   └── middleware/         # 中介軟體
│       └── auth_middleware.go # 驗證中介軟體
├── pkg/                    # 公用元件（工具、輔助功能等）
│   ├── logger/             # 日誌記錄
│   │   └── logger.go       # 日誌管理
│   └── utils/              # 公用工具
│       └── utils.go        # 各類工具函數
├── .env                    # 環境變數配置
├── go.mod                  # Go modules 配置
└── README.md               # 專案說明文件

```