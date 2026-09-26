package services

import (
	"errors"
	"github.com/q90016200/scheduler-go/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var ErrUserDisabled = errors.New("帳號已停用")

type AuthService struct {
	db *gorm.DB
}

func NewAuthService(db *gorm.DB) *AuthService {
	return &AuthService{db: db}
}

func (s *AuthService) Authenticate(username, password string) (*models.User, error) {
	var user models.User
	if err := s.db.Where("user_name = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, err
	}
	if user.Status == models.STATUS_DISABLED {
		return nil, ErrUserDisabled
	}
	return &user, nil
}
