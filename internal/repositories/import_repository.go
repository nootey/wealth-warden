package repositories

import (
	"time"
	"wealth-warden/internal/models"

	"gorm.io/gorm"
)

type ImportRepository struct {
	DB *gorm.DB
}

func NewImportRepository(db *gorm.DB) *ImportRepository {
	return &ImportRepository{DB: db}
}

func (r *ImportRepository) FindImportsByImportType(tx *gorm.DB, userID int64, importType string) ([]models.Import, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	var records []models.Import

	q := db.Model(&models.Import{}).
		Where("user_id = ? AND import_type = ?", userID, importType)

	err := q.Find(&records).Error
	if err != nil {
		return nil, err
	}

	return records, nil
}

func (r *ImportRepository) FindImportByID(tx *gorm.DB, id, userID int64, importType string) (*models.Import, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	var record models.Import

	q := db.Model(&models.Import{}).
		Where("id= ? AND user_id = ? AND import_type = ?", id, userID, importType)

	err := q.First(&record).Error
	if err != nil {
		return nil, err
	}

	return &record, nil
}

func (r *ImportRepository) InsertImport(tx *gorm.DB, record models.Import) (int64, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	if err := db.Create(&record).Error; err != nil {
		return 0, err
	}
	return record.ID, nil
}

func (r *ImportRepository) UpdateImport(tx *gorm.DB, id int64, fields map[string]interface{}) error {
	db := tx
	if db == nil {
		db = r.DB
	}
	return db.Model(&models.Import{}).Where("id = ?", id).Updates(fields).Error
}

func (r *ImportRepository) DeleteImport(tx *gorm.DB, id, userID int64) error {
	db := tx
	if db == nil {
		db = r.DB
	}

	if err := db.Where("id = ? AND user_id = ?", id, userID).
		Delete(&models.Import{}).Error; err != nil {
		return err
	}

	return nil
}

type TxnDelta struct {
	AccountID int64     `gorm:"column:account_id"`
	AsOf      time.Time `gorm:"column:as_of"`
	Inflows   string    `gorm:"column:inflows"`  // numeric as text
	Outflows  string    `gorm:"column:outflows"` // numeric as text
}

func (r *ImportRepository) AggregateImportTxnDeltas(
	tx *gorm.DB, userID, importID int64,
) ([]TxnDelta, error) {
	db := tx
	if db == nil {
		db = r.DB
	}
	var rows []TxnDelta
	err := db.Raw(`
		SELECT
			t.account_id,
			t.txn_date::date AS as_of,
			COALESCE(SUM(CASE WHEN LOWER(t.transaction_type) <> 'expense' THEN t.amount ELSE 0 END), 0)::text AS inflows,
			COALESCE(SUM(CASE WHEN LOWER(t.transaction_type) =  'expense' THEN t.amount ELSE 0 END), 0)::text AS outflows
		FROM transactions t
		JOIN accounts a ON a.id = t.account_id AND a.user_id = ?
		WHERE t.import_id = ? AND t.deleted_at IS NULL
		GROUP BY t.account_id, t.txn_date::date
		ORDER BY t.account_id, t.txn_date::date
	`, userID, importID).Scan(&rows).Error
	return rows, err
}

type TrDelta struct {
	FromAccountID int64     `gorm:"column:from_account_id"`
	ToAccountID   int64     `gorm:"column:to_account_id"`
	AsOf          time.Time `gorm:"column:as_of"`
	Amount        string    `gorm:"column:amount"` // numeric as text
}

func (r *ImportRepository) AggregateImportTransferDeltas(
	tx *gorm.DB, userID, importID int64,
) ([]TrDelta, error) {
	db := tx
	if db == nil {
		db = r.DB
	}
	var rows []TrDelta
	err := db.Raw(`
		SELECT
			tr.from_account_id,
			tr.to_account_id,
			tr.txn_date::date AS as_of,
			COALESCE(SUM(tr.amount), 0)::text AS amount
		FROM transfers tr
		JOIN accounts af ON af.id = tr.from_account_id AND af.user_id = ?
		JOIN accounts at ON at.id = tr.to_account_id   AND at.user_id = ?
		WHERE tr.import_id = ? AND tr.deleted_at IS NULL
		GROUP BY tr.from_account_id, tr.to_account_id, tr.txn_date::date
		ORDER BY tr.txn_date::date
	`, userID, userID, importID).Scan(&rows).Error
	return rows, err
}

// Optional: earliest impacted date per account (transactions + transfers)
type MinImpact struct {
	AccountID int64     `gorm:"column:account_id"`
	MinAsOf   time.Time `gorm:"column:min_as_of"`
}

func (r *ImportRepository) EarliestImpactPerAccount(
	tx *gorm.DB, userID, importID int64,
) ([]MinImpact, error) {
	db := tx
	if db == nil {
		db = r.DB
	}
	var rows []MinImpact
	err := db.Raw(`
		WITH t AS (
			SELECT t.account_id, MIN(t.txn_date::date) AS d
			FROM transactions t
			JOIN accounts a ON a.id = t.account_id AND a.user_id = ?
			WHERE t.import_id = ? AND t.deleted_at IS NULL
			GROUP BY t.account_id
		),
		tr AS (
			SELECT x.account_id, MIN(x.as_of) AS d
			FROM (
				SELECT tr.from_account_id AS account_id, MIN(tr.txn_date::date) AS as_of
				FROM transfers tr JOIN accounts a ON a.id = tr.from_account_id AND a.user_id = ?
				WHERE tr.import_id = ? AND tr.deleted_at IS NULL
				GROUP BY tr.from_account_id
				UNION ALL
				SELECT tr.to_account_id   AS account_id, MIN(tr.txn_date::date) AS as_of
				FROM transfers tr JOIN accounts a ON a.id = tr.to_account_id   AND a.user_id = ?
				WHERE tr.import_id = ? AND tr.deleted_at IS NULL
				GROUP BY tr.to_account_id
			) x
			GROUP BY x.account_id
		)
		SELECT COALESCE(t.account_id, tr.account_id) AS account_id,
			   LEAST(COALESCE(t.d, 'infinity'), COALESCE(tr.d, 'infinity')) AS min_as_of
		FROM t
		FULL JOIN tr ON tr.account_id = t.account_id
		ORDER BY 1
	`, userID, importID, userID, importID, userID, importID).Scan(&rows).Error
	return rows, err
}
