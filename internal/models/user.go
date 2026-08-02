package models

type User struct {
	ID        string `json:"id" gorm:"primaryKey;size:20"`
	UserName  string `json:"user_name" gorm:"unique;size:20"`
	Password  string `json:"password" gorm:"size:100;not null"`
	CreatedAt int64  `json:"created_at" gorm:"autoCreateTime;<-:create"`
	UpdatedAt int64  `json:"updated_at" gorm:"autoUpdateTime"`
	Status    bool   `json:"status" gorm:"not null;index:idx_status"` // 用戶狀態，true 為啟用
}
