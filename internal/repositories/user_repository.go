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

func (r *UserRepository) GetUserByID(id uint, includeSecrets bool) (*models.User, error) {
	var user models.User

	// Preload the user's global Role and organization membership details.
	query := r.DB.
		Preload("Role").
		Preload("Organizations.Organization")
	if includeSecrets {
		query = query.Preload("Secrets")
	}

	err := query.First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetUserByEmail(email string, includeSecrets bool) (*models.User, error) {
	var user models.User

	query := r.DB.
		Preload("Role").
		Preload("PrimaryOrganization")
	if includeSecrets {
		query = query.Preload("Secrets")
	}

	err := query.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetAllUsers() ([]models.User, error) {
	var users []models.User

	// Preload the organizations for each user, including the Organization and organization-specific Role.
	err := r.DB.
		Omit("Secrets").
		Preload("PrimaryOrganization").
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

func (r *UserRepository) DeleteUser(id uint) error {
	return r.DB.Delete(&models.User{}, id).Error
}
