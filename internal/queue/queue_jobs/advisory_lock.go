package queue_jobs

import (
	"context"
	"database/sql/driver"
	"errors"
	"time"

	"gorm.io/gorm"
)

// LockKeyInvestmentRebuild is shared by every job that clears a user's investment
// cash flows and snapshots and rebuilds them. Those jobs collide with each other,
// not only with themselves: two runs that both clear before either adds count
// each trade twice.
const LockKeyInvestmentRebuild int64 = 8110001

// ErrRebuildLockHeld keeps a skipped run on the queue's retry path. Returning nil
// would delete the job, so a rebuild someone asked for would report success
// having done nothing.
var ErrRebuildLockHeld = errors.New("another investment rebuild holds the lock")

type JobLock interface {
	TryLock(ctx context.Context, key int64) (release func(), acquired bool, err error)
}

type AdvisoryLock struct {
	db *gorm.DB
}

func NewAdvisoryLock(db *gorm.DB) *AdvisoryLock {
	return &AdvisoryLock{db: db}
}

// TryLock pins one pooled connection and holds the lock on it, since a job opens
// many short transactions and a transaction-scoped lock would not span the run.
func (l *AdvisoryLock) TryLock(ctx context.Context, key int64) (func(), bool, error) {
	sqlDB, err := l.db.DB()
	if err != nil {
		return nil, false, err
	}

	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return nil, false, err
	}

	var acquired bool
	if err := conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", key).Scan(&acquired); err != nil {
		_ = conn.Close()
		return nil, false, err
	}
	if !acquired {
		_ = conn.Close()
		return nil, false, nil
	}

	release := func() {
		// The job's context is usually cancelled by the time we unlock.
		relCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if _, err := conn.ExecContext(relCtx, "SELECT pg_advisory_unlock($1)", key); err != nil {
			// A connection that may still hold the lock must not go back to the
			// pool; the leaked lock would block the job for good.
			_ = conn.Raw(func(any) error { return driver.ErrBadConn })
		}
		_ = conn.Close()
	}

	return release, true, nil
}
