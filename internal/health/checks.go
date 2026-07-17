package health

import (
	"context"
	"database/sql"

	"github.com/redis/go-redis/v9"
)

type DBChecker struct {
	db *sql.DB
}

func NewDBChecker(db *sql.DB) *DBChecker {
	return &DBChecker{db: db}
}

func (d *DBChecker) Name() string                    { return "db" }
func (d *DBChecker) Check(ctx context.Context) error { return d.db.PingContext(ctx) }

type RedisChecker struct {
	rdb *redis.Client
}

func NewRedisChecker(rdb *redis.Client) *RedisChecker {
	return &RedisChecker{rdb: rdb}
}

func (r *RedisChecker) Name() string                    { return "redis" }
func (r *RedisChecker) Check(ctx context.Context) error { return r.rdb.Ping(ctx).Err() }
