package repositories

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/pkg/utils"
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

func (r *UserRepository) FindUserByID(tx *gorm.DB, id int64) (*models.User, error) {

	db := tx
	if db == nil {
		db = r.DB
	}

	var record models.User

	query := r.DB

	err := query.
		Preload("Role.Permissions").
		Where("id = ?", id).
		First(&record).Error
	if err != nil {
		return nil, err
	}

	return &record, nil
}

func (r *UserRepository) FindUserByEmail(tx *gorm.DB, email string) (*models.User, error) {

	db := tx
	if db == nil {
		db = r.DB
	}

	var record models.User

	query := r.DB

	err := query.
		Preload("Role.Permissions").
		Where("email = ?", email).
		First(&record).Error
	if err != nil {
		return nil, err
	}

	return &record, nil
}

func (r *UserRepository) FindUserInvitationByHash(tx *gorm.DB, hash string) (*models.Invitation, error) {

	db := tx
	if db == nil {
		db = r.DB
	}

	var record models.Invitation

	err := r.DB.Where("hash =?", hash).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *UserRepository) FindTokenByValue(tx *gorm.DB, tokenType, tokenValue string) (*models.Token, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	var record models.Token
	err := r.DB.
		Where("token_type = ? AND token_value = ?", tokenType, tokenValue).
		First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *UserRepository) FindTokenByData(tx *gorm.DB, tokenType string, dataIndex string, dataValue interface{}) (*models.Token, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	fragment := datatypes.JSONMap{dataIndex: dataValue}

	var record models.Token

	err := r.DB.
		Where("token_type = ? AND data @> ?", tokenType, fragment).
		First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
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

func (r *UserRepository) InsertInvitation(tx *gorm.DB, record *models.Invitation) (int64, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	if err := db.Create(&record).Error; err != nil {
		return 0, err
	}
	return record.ID, nil
}

func (r *UserRepository) InsertToken(tx *gorm.DB, tokenType string, dataIndex string, dataValue interface{}) (*models.Token, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	randomToken, err := utils.GenerateRandomToken(32)
	if err != nil {
		return nil, err
	}

	record := models.Token{
		TokenType:  tokenType,
		TokenValue: randomToken,
		Data: datatypes.JSONMap{
			dataIndex: dataValue,
		},
	}

	if err := db.Create(&record).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *UserRepository) InsertUser(tx *gorm.DB, record *models.User) (int64, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	if err := db.Create(&record).Error; err != nil {
		return 0, err
	}
	return record.ID, nil
}

func (r *UserRepository) UpdateUser(tx *gorm.DB, record models.User) (int64, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	if err := db.Model(models.User{}).
		Where("id = ?", record.ID).
		Updates(map[string]interface{}{
			"password":   record.Password,
			"updated_at": time.Now(),
		}).Error; err != nil {
		return 0, err
	}
	return record.ID, nil
}

func (r *UserRepository) DeleteUser(id int64) error {
	return r.DB.Delete(&models.User{}, id).Error
}

func (r *UserRepository) DeleteTokenByData(tx *gorm.DB, tokenType, dataIndex string, dataValue interface{}) error {
	db := tx
	if db == nil {
		db = r.DB
	}

	fragment := datatypes.JSONMap{dataIndex: dataValue}

	return db.
		Where("token_type = ? AND data @> ?", tokenType, fragment).
		Delete(&models.Token{}).Error
}
