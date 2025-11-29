package repositories

import (
	"context"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/pkg/utils"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type UserRepositoryInterface interface {
	BeginTx(ctx context.Context) (*gorm.DB, error)
	FindUsers(ctx context.Context, tx *gorm.DB, offset, limit int, sortField, sortOrder string, filters []utils.Filter, includeDeleted bool) ([]models.User, error)
	CountUsers(ctx context.Context, tx *gorm.DB, filters []utils.Filter, includeDeleted bool) (int64, error)
	FindInvitations(ctx context.Context, tx *gorm.DB, offset, limit int, sortField, sortOrder string, filters []utils.Filter) ([]models.Invitation, error)
	CountInvitations(ctx context.Context, tx *gorm.DB, filters []utils.Filter) (int64, error)
	GetPasswordByEmail(ctx context.Context, tx *gorm.DB, email string) (string, error)
	FindUserByID(ctx context.Context, tx *gorm.DB, id int64) (*models.User, error)
	FindUserByEmail(ctx context.Context, tx *gorm.DB, email string) (*models.User, error)
	FindInvitationByID(ctx context.Context, tx *gorm.DB, id int64) (*models.Invitation, error)
	FindUserInvitationByHash(ctx context.Context, tx *gorm.DB, hash string) (*models.Invitation, error)
	FindTokenByValue(ctx context.Context, tx *gorm.DB, tokenType, tokenValue string) (*models.Token, error)
	FindTokenByData(ctx context.Context, tx *gorm.DB, tokenType string, dataIndex string, dataValue interface{}) (*models.Token, error)
	GetAllUsers(ctx context.Context, tx *gorm.DB) ([]models.User, error)
	InsertInvitation(ctx context.Context, tx *gorm.DB, record *models.Invitation) (int64, error)
	InsertToken(ctx context.Context, tx *gorm.DB, tokenType string, dataIndex string, dataValue interface{}) (*models.Token, error)
	InsertUser(ctx context.Context, tx *gorm.DB, record *models.User) (int64, error)
	UpdateUser(ctx context.Context, tx *gorm.DB, record models.User) (int64, error)
	UpdateUserPassword(ctx context.Context, tx *gorm.DB, id int64, password string) error
	DeleteUser(ctx context.Context, tx *gorm.DB, id int64) error
	DeleteInvitation(ctx context.Context, tx *gorm.DB, id int64) error
	DeleteTokenByData(ctx context.Context, tx *gorm.DB, tokenType, dataIndex string, dataValue interface{}) error
}

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

var _ UserRepositoryInterface = (*UserRepository)(nil)

func (r *UserRepository) BeginTx(ctx context.Context) (*gorm.DB, error) {
	tx := r.db.WithContext(ctx).Begin()
	return tx, tx.Error
}

func (r *UserRepository) FindUsers(ctx context.Context, tx *gorm.DB, offset, limit int, sortField, sortOrder string, filters []utils.Filter, includeDeleted bool) ([]models.User, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var records []models.User
	q := db.Model(&models.User{}).
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

func (r *UserRepository) CountUsers(ctx context.Context, tx *gorm.DB, filters []utils.Filter, includeDeleted bool) (int64, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var totalRecords int64
	q := db.Model(&models.User{}).
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

func (r *UserRepository) FindInvitations(ctx context.Context, tx *gorm.DB, offset, limit int, sortField, sortOrder string, filters []utils.Filter) ([]models.Invitation, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var records []models.Invitation
	q := db.Model(&models.Invitation{}).
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

func (r *UserRepository) CountInvitations(ctx context.Context, tx *gorm.DB, filters []utils.Filter) (int64, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var totalRecords int64
	q := db.Model(&models.Invitation{}).
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

func (r *UserRepository) GetPasswordByEmail(ctx context.Context, tx *gorm.DB, email string) (string, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var password string
	err := db.Model(&models.User{}).Select("password").Where("email = ?", email).Scan(&password).Error
	if err != nil {
		return "", err
	}
	return password, nil
}

func (r *UserRepository) FindUserByID(ctx context.Context, tx *gorm.DB, id int64) (*models.User, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var record models.User
	err := db.Preload("Role.Permissions").
		Where("id = ?", id).
		First(&record).Error
	if err != nil {
		return nil, err
	}

	return &record, nil
}

func (r *UserRepository) FindUserByEmail(ctx context.Context, tx *gorm.DB, email string) (*models.User, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var record models.User
	err := db.Preload("Role.Permissions").
		Where("email = ?", email).
		First(&record).Error
	if err != nil {
		return nil, err
	}

	return &record, nil
}

func (r *UserRepository) FindInvitationByID(ctx context.Context, tx *gorm.DB, id int64) (*models.Invitation, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var record models.Invitation
	err := db.Where("id =?", id).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *UserRepository) FindUserInvitationByHash(ctx context.Context, tx *gorm.DB, hash string) (*models.Invitation, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var record models.Invitation
	err := db.Where("hash =?", hash).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *UserRepository) FindTokenByValue(ctx context.Context, tx *gorm.DB, tokenType, tokenValue string) (*models.Token, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var record models.Token
	err := db.Where("token_type = ? AND token_value = ?", tokenType, tokenValue).
		First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *UserRepository) FindTokenByData(ctx context.Context, tx *gorm.DB, tokenType string, dataIndex string, dataValue interface{}) (*models.Token, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	fragment := datatypes.JSONMap{dataIndex: dataValue}
	var record models.Token

	err := db.Where("token_type = ? AND data @> ?", tokenType, fragment).
		First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *UserRepository) GetAllUsers(ctx context.Context, tx *gorm.DB) ([]models.User, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var users []models.User
	err := r.db.Preload("Role").
		Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (r *UserRepository) InsertInvitation(ctx context.Context, tx *gorm.DB, record *models.Invitation) (int64, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	if err := db.Create(&record).Error; err != nil {
		return 0, err
	}
	return record.ID, nil
}

func (r *UserRepository) InsertToken(ctx context.Context, tx *gorm.DB, tokenType string, dataIndex string, dataValue interface{}) (*models.Token, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

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

func (r *UserRepository) InsertUser(ctx context.Context, tx *gorm.DB, record *models.User) (int64, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	if err := db.Create(&record).Error; err != nil {
		return 0, err
	}
	return record.ID, nil
}

func (r *UserRepository) UpdateUser(ctx context.Context, tx *gorm.DB, record models.User) (int64, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

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

func (r *UserRepository) UpdateUserPassword(ctx context.Context, tx *gorm.DB, id int64, password string) error {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

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

func (r *UserRepository) DeleteUser(ctx context.Context, tx *gorm.DB, id int64) error {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

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

func (r *UserRepository) DeleteInvitation(ctx context.Context, tx *gorm.DB, id int64) error {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	return db.Where("id = ?", id).
		Delete(&models.Invitation{}).Error
}

func (r *UserRepository) DeleteTokenByData(ctx context.Context, tx *gorm.DB, tokenType, dataIndex string, dataValue interface{}) error {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	fragment := datatypes.JSONMap{dataIndex: dataValue}
	return db.Where("token_type = ? AND data @> ?", tokenType, fragment).
		Delete(&models.Token{}).Error
}
