package models

import (
	"time"
)

type User struct {
	ID        string    `json:"id" gorm:"primaryKey;size=20"`
	UserName  string    `json:"user_name" gorm:"unique;size:20"`
	Password  string    `json:"password" gorm:"size:100;not null"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	Status    bool      `json:"status" gorm:"not null;index:idx_status"` //用戶狀態  'active', 'inactive'
}
