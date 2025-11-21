package repositories

import (
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/pkg/utils"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type UserRepository struct {
	DB *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{DB: db}
}

func (r *UserRepository) FindUsers(offset, limit int, sortField, sortOrder string, filters []utils.Filter, includeDeleted bool) ([]models.User, error) {

	var records []models.User

	q := r.DB.Model(&models.User{}).
		Preload("Role").
		Joins("JOIN roles ON roles.id = users.role_id").
		Where("roles.name != ?", "super-admin")

	if !includeDeleted {
		q = q.Where("users.deleted_at IS NULL")
	}

	joins := utils.GetRequiredJoins(filters)
	orderBy := utils.ConstructOrderByClause(&joins, "users", sortField, sortOrder)

	for _, join := range joins {
		q = q.Joins(join)
	}

	q = utils.ApplyFilters(q, filters)

	err := q.
		Order(orderBy).
		Limit(limit).
		Offset(offset).
		Find(&records).Error
	if err != nil {
		return nil, err
	}

	return records, nil
}

func (r *UserRepository) CountUsers(filters []utils.Filter, includeDeleted bool) (int64, error) {
	var totalRecords int64

	q := r.DB.Model(&models.User{}).
		Preload("Role").
		Joins("JOIN roles ON roles.id = users.role_id").
		Where("roles.name != ?", "super-admin")

	if !includeDeleted {
		q = q.Where("users.deleted_at IS NULL")
	}

	joins := utils.GetRequiredJoins(filters)
	for _, join := range joins {
		q = q.Joins(join)
	}

	q = utils.ApplyFilters(q, filters)

	err := q.Count(&totalRecords).Error
	if err != nil {
		return 0, err
	}
	return totalRecords, nil
}

func (r *UserRepository) FindInvitations(offset, limit int, sortField, sortOrder string, filters []utils.Filter) ([]models.Invitation, error) {

	var records []models.Invitation

	q := r.DB.Model(&models.Invitation{}).
		Preload("Role")

	joins := utils.GetRequiredJoins(filters)
	orderBy := utils.ConstructOrderByClause(&joins, "invitations", sortField, sortOrder)

	for _, join := range joins {
		q = q.Joins(join)
	}

	q = utils.ApplyFilters(q, filters)

	err := q.
		Order(orderBy).
		Limit(limit).
		Offset(offset).
		Find(&records).Error
	if err != nil {
		return nil, err
	}

	return records, nil
}

func (r *UserRepository) CountInvitations(filters []utils.Filter) (int64, error) {
	var totalRecords int64

	q := r.DB.Model(&models.Invitation{}).
		Preload("Role")

	joins := utils.GetRequiredJoins(filters)
	for _, join := range joins {
		q = q.Joins(join)
	}

	q = utils.ApplyFilters(q, filters)

	err := q.Count(&totalRecords).Error
	if err != nil {
		return 0, err
	}
	return totalRecords, nil
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
	err := db.
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
	err := db.
		Preload("Role.Permissions").
		Where("email = ?", email).
		First(&record).Error
	if err != nil {
		return nil, err
	}

	return &record, nil
}

func (r *UserRepository) FindInvitationByID(tx *gorm.DB, id int64) (*models.Invitation, error) {

	db := tx
	if db == nil {
		db = r.DB
	}

	var record models.Invitation

	err := db.Where("id =?", id).First(&record).Error
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

	err := db.Where("hash =?", hash).First(&record).Error
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
	err := db.
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

	err := db.
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
	
	updates := map[string]interface{}{
		"display_name": record.DisplayName,
		"role_id":      record.RoleID,
		"updated_at":   time.Now().UTC(),
	}

	if record.Email != "" {
		updates["email"] = record.Email
	}

	if err := db.Model(models.User{}).
		Where("id = ?", record.ID).
		Updates(updates).Error; err != nil {
		return 0, err
	}

	return record.ID, nil
}

func (r *UserRepository) UpdateUserPassword(tx *gorm.DB, id int64, password string) error {
	db := tx
	if db == nil {
		db = r.DB
	}

	if err := db.Model(models.User{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"password":   password,
			"updated_at": time.Now().UTC(),
		}).Error; err != nil {
		return err
	}
	return nil
}

func (r *UserRepository) DeleteUser(tx *gorm.DB, id int64) error {
	db := tx
	if db == nil {
		db = r.DB
	}

	res := db.Model(&models.User{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]any{
			"deleted_at": time.Now().UTC(),
			"updated_at": time.Now().UTC(),
		})

	if res.Error != nil {
		return res.Error
	}
	return nil
}

func (r *UserRepository) DeleteInvitation(tx *gorm.DB, id int64) error {
	db := tx
	if db == nil {
		db = r.DB
	}

	return db.Where("id = ?", id).
		Delete(&models.Invitation{}).Error
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
