package services

import (
	"github.com/q90016200/scheduler-go/internal/models"
	"github.com/rs/xid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService struct {
	db *gorm.DB
}

func NewUserService(db *gorm.DB) *UserService {
	return &UserService{db: db}
}

// ListUsers 依 name 模糊比對篩選（可留空查全部），limit/offset 分頁
func (s *UserService) ListUsers(name string, limit, offset int) ([]models.User, int64, error) {
	var users []models.User
	query := s.db.Model(&models.User{})
	if name != "" {
		query = query.Where("user_name ILIKE ?", "%"+name+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Limit(limit).Offset(offset).Find(&users).Error
	for i := range users {
		users[i].Password = ""
	}
	return users, total, err
}

func (s *UserService) GetUser(id string) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, "id = ?", id).Error; err != nil {
		return nil, err
	}
	user.Password = ""
	return &user, nil
}

func (s *UserService) CreateUser(input models.CreateUserReq) (*models.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	status := input.Status
	if status == "" {
		status = models.STATUS_ENABLED
	}

	user := &models.User{
		ID:       xid.New().String(),
		UserName: input.UserName,
		Password: string(hashedPassword),
		Status:   status,
	}
	if err := s.db.Create(user).Error; err != nil {
		return nil, err
	}

	user.Password = ""
	return user, nil
}

// UpdateUser Password 為空字串時不更改密碼
func (s *UserService) UpdateUser(id string, input models.UpdateUserReq) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, "id = ?", id).Error; err != nil {
		return nil, err
	}

	user.UserName = input.UserName
	if input.Status != "" {
		user.Status = input.Status
	}
	if input.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		user.Password = string(hashedPassword)
	}

	if err := s.db.Save(&user).Error; err != nil {
		return nil, err
	}

	user.Password = ""
	return &user, nil
}

func (s *UserService) UpdateUserStatus(id, status string) error {
	return s.db.Model(&models.User{}).Where("id = ?", id).Update("status", status).Error
}

func (s *UserService) DeleteUser(id string) error {
	result := s.db.Where("id = ?", id).Delete(&models.User{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
