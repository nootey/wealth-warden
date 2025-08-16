package repositories

import (
	"gorm.io/gorm"
	"wealth-warden/internal/models"
)

type UserRepository struct {
	DB *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{DB: db}
}

func (r *UserRepository) GetPasswordByEmail(email string) (string, error) {
	var password string
	err := r.DB.Model(&models.User{}).Select("password").Where("email = ?", email).Scan(&password).Error
	if err != nil {
		return "", err
	}
	return password, nil
}

func (r *UserRepository) GetUserByID(id int64) (*models.User, error) {
	var user models.User

	query := r.DB.
		Preload("Role")

	err := query.First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetUserByEmail(email string) (*models.User, error) {
	var user models.User

	query := r.DB.
		Preload("Role")

	err := query.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetAllUsers() ([]models.User, error) {
	var users []models.User

	err := r.DB.
		Preload("Role").
		Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (r *UserRepository) CreateUser(user *models.User) error {
	return r.DB.Create(user).Error
}

func (r *UserRepository) UpdateUser(user *models.User) error {
	return r.DB.Save(user).Error
}

func (r *UserRepository) DeleteUser(id int64) error {
	return r.DB.Delete(&models.User{}, id).Error
}
