package models

type User struct {
	ID        string `json:"id" gorm:"primaryKey;size:20"`
	UserName  string `json:"user_name" gorm:"unique;size:20"`
	Password  string `json:"password" gorm:"size:100;not null"`
	CreatedAt int64  `json:"created_at" gorm:"autoCreateTime;<-:create"`
	UpdatedAt int64  `json:"updated_at" gorm:"autoUpdateTime"`
	Status    string `json:"status" gorm:"column:status;size:10;index:idx_status"` // 用戶狀態 ('enabled', 'disabled')
}

type CreateUserReq struct {
	UserName string `json:"user_name" validate:"required"`
	Password string `json:"password" validate:"required"`
	Status   string `json:"status" validate:"omitempty,oneof=enabled disabled"`
}

// UpdateUserReq Password 留空代表不更改密碼
type UpdateUserReq struct {
	UserName string `json:"user_name" validate:"required"`
	Password string `json:"password"`
	Status   string `json:"status" validate:"omitempty,oneof=enabled disabled"`
}

type UpdateUserStatusReq struct {
	Status string `json:"status" validate:"required,oneof=enabled disabled"`
}
