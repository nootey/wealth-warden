package repositories

import (
	"gorm.io/gorm"
	"wealth-warden/internal/models"
	"wealth-warden/pkg/utils"
)

type AccountRepository struct {
	DB *gorm.DB
}

func NewAccountRepository(db *gorm.DB) *AccountRepository {
	return &AccountRepository{DB: db}
}

func (r *AccountRepository) FindAccounts(user *models.User, year, offset, limit int, sortField, sortOrder string, filters []utils.Filter) ([]models.Account, error) {

	var records []models.Account

	query := r.DB.
		Preload("AccountType").
		Preload("Balance").
		Where("accounts.user_id = ?", user.ID)

	joins := utils.GetRequiredJoins(filters)
	orderBy := utils.ConstructOrderByClause(&joins, sortField, sortOrder)

	for _, join := range joins {
		query = query.Joins(join)
	}

	query = utils.ApplyFilters(query, filters)

	err := query.
		Order(orderBy).
		Limit(limit).
		Offset(offset).
		Find(&records).Error
	if err != nil {
		return nil, err
	}

	return records, nil
}

func (r *AccountRepository) CountAccounts(user *models.User, year int, filters []utils.Filter) (int64, error) {
	var totalRecords int64

	query := r.DB.Model(&models.Account{}).
		Where("accounts.user_id = ?", user.ID)

	joins := utils.GetRequiredJoins(filters)
	for _, join := range joins {
		query = query.Joins(join)
	}

	query = utils.ApplyFilters(query, filters)

	err := query.Count(&totalRecords).Error
	if err != nil {
		return 0, err
	}
	return totalRecords, nil
}

func (r *AccountRepository) FindAllAccounts(user *models.User) ([]models.Account, error) {
	var records []models.Account
	result := r.DB.Find(&records).Where("accounts.user_id = ?", user.ID)
	return records, result.Error
}

func (r *AccountRepository) FindAllAccountTypes(user *models.User) ([]models.AccountType, error) {
	var records []models.AccountType
	result := r.DB.Find(&records)
	return records, result.Error
}

func (r *AccountRepository) FindAccountByID(ID, userID uint) (models.Account, error) {
	var record models.Account
	result := r.DB.Where("id = ? AND user_id = ?", ID, userID).First(&record)
	return record, result.Error
}

func (r *AccountRepository) FindAccountTypeByID(ID uint) (models.AccountType, error) {
	var record models.AccountType
	result := r.DB.Where("id = ?", ID).First(&record)
func (r *AccountRepository) FindBalanceForAccountID(tx *gorm.DB, accID uint) (models.Balance, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	var record models.Balance
	result := db.Where("account_id = ?", accID).First(&record)
	return record, result.Error
}

func (r *AccountRepository) InsertAccount(tx *gorm.DB, newRecord *models.Account) (uint, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	if err := db.Create(&newRecord).Error; err != nil {
		return 0, err
	}
	return newRecord.ID, nil
}

func (r *AccountRepository) InsertBalance(tx *gorm.DB, newRecord *models.Balance) (uint, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	if err := db.Create(&newRecord).Error; err != nil {
		return 0, err
	}
	return newRecord.ID, nil
}

func (r *AccountRepository) UpdateBalance(tx *gorm.DB, record models.Balance) (uint, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	if err := db.Model(models.Balance{}).
		Where("id = ?", record.ID).
		Updates(record).Error; err != nil {
		return 0, err
	}
	return record.ID, nil
}
