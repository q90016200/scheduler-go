package services

import (
	"errors"
	"github.com/q90016200/scheduler-go/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var ErrUserDisabled = errors.New("帳號已停用")

// ErrInvalidCredentials 使用者不存在或密碼不符；兩者刻意不區分，避免洩漏帳號是否存在。
// 其他錯誤（DB 連線等）原樣回傳，讓 controller 記 log 回 500，不被當成「密碼錯誤」吞掉
var ErrInvalidCredentials = errors.New("invalid credentials")

type AuthService struct {
	db *gorm.DB
}

func NewAuthService(db *gorm.DB) *AuthService {
	return &AuthService{db: db}
}

func (s *AuthService) Authenticate(username, password string) (*models.User, error) {
	var user models.User
	if err := s.db.Where("user_name = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if user.Status == models.STATUS_DISABLED {
		return nil, ErrUserDisabled
	}
	return &user, nil
}
