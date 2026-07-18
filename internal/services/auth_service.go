package services

import (
	"github.com/q90016200/scheduler-go/internal/models"
	"github.com/rs/xid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"time"
)

type AuthService struct {
	db *gorm.DB
}

func NewAuthService(db *gorm.DB) *AuthService {
	return &AuthService{db: db}
}

func (s *AuthService) Register(username, password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user := &models.User{
		ID:        xid.New().String(),
		UserName:  username,
		Password:  string(hashedPassword),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Status:    true,
	}
	return s.db.Create(user).Error
}

func (s *AuthService) Authenticate(username, password string) (*models.User, error) {
	var user models.User
	if err := s.db.Where("user_name = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	return &user, err
}
