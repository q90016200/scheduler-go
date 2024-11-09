package services

import (
	"github.com/q90016200/schedule-go/internal/models"
	"github.com/q90016200/schedule-go/internal/repositories"
	"github.com/rs/xid"
	"golang.org/x/crypto/bcrypt"
	"time"
)

type AuthService struct {
	repo *repositories.RepositoryManager
}

func NewAuthService(repo *repositories.RepositoryManager) *AuthService {
	return &AuthService{repo: repo}
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
	return s.repo.UserRepo.Create(user)
}

func (s *AuthService) Authenticate(username, password string) (*models.User, error) {
	user, err := s.repo.UserRepo.FindByUsername(username)
	if err != nil {
		return nil, err
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	return user, err
}
