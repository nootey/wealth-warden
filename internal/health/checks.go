package health

import (
	"context"
	"database/sql"
)

type DBChecker struct {
	db *sql.DB
}

func NewDBChecker(db *sql.DB) *DBChecker {
	return &DBChecker{db: db}
}

func (d *DBChecker) Name() string                    { return "db" }
func (d *DBChecker) Check(ctx context.Context) error { return d.db.PingContext(ctx) }
