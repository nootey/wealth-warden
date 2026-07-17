package sessions

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/pkg/config"

	"github.com/redis/go-redis/v9"
)

const (
	CookieName = "session"

	// last_seen is only bumped when older than this, to keep validation cheap.
	lastSeenInterval = time.Hour
)

var ErrNotFound = errors.New("session not found")

type Store struct {
	rdb           *redis.Client
	ttl           time.Duration
	rememberMeTTL time.Duration
	maxLifetime   time.Duration
}

func NewStore(rdb *redis.Client, cfg config.SessionConfig) *Store {
	return &Store{
		rdb:           rdb,
		ttl:           time.Duration(cfg.TTLHours) * time.Hour,
		rememberMeTTL: time.Duration(cfg.RememberMeTTLHours) * time.Hour,
		maxLifetime:   time.Duration(cfg.MaxLifetimeHours) * time.Hour,
	}
}

func (s *Store) TTL(rememberMe bool) time.Duration {
	if rememberMe {
		return s.rememberMeTTL
	}
	return s.ttl
}

func sessionKey(id string) string {
	return "session:" + id
}

func userKey(userID int64) string {
	return fmt.Sprintf("user_sessions:%d", userID)
}

func (s *Store) Create(ctx context.Context, userID int64, rememberMe bool, userAgent, ip string) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	id := base64.RawURLEncoding.EncodeToString(raw)

	now := time.Now().Unix()
	remember := 0
	if rememberMe {
		remember = 1
	}

	pipe := s.rdb.TxPipeline()
	pipe.HSet(ctx, sessionKey(id),
		"user_id", userID,
		"created_at", now,
		"last_seen", now,
		"remember_me", remember,
		"user_agent", userAgent,
		"ip", ip,
	)
	pipe.Expire(ctx, sessionKey(id), s.TTL(rememberMe))
	pipe.SAdd(ctx, userKey(userID), id)
	pipe.Expire(ctx, userKey(userID), s.maxLifetime)
	if _, err := pipe.Exec(ctx); err != nil {
		return "", err
	}

	return id, nil
}

// Validate resolves a session ID to its user, enforcing the absolute max lifetime.
// Expiry is fixed at login (matching the cookie's Max-Age); last_seen is bumped
// best-effort for the sessions listing.
func (s *Store) Validate(ctx context.Context, id string) (int64, error) {
	fields, err := s.rdb.HGetAll(ctx, sessionKey(id)).Result()
	if err != nil {
		return 0, err
	}
	if len(fields) == 0 {
		return 0, ErrNotFound
	}

	userID, err := strconv.ParseInt(fields["user_id"], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("malformed session: %w", err)
	}

	createdAt, err := strconv.ParseInt(fields["created_at"], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("malformed session: %w", err)
	}
	if time.Since(time.Unix(createdAt, 0)) > s.maxLifetime {
		_ = s.Delete(ctx, id)
		return 0, ErrNotFound
	}

	lastSeen, err := strconv.ParseInt(fields["last_seen"], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("malformed session: %w", err)
	}
	if time.Since(time.Unix(lastSeen, 0)) >= lastSeenInterval {
		_ = s.rdb.HSet(ctx, sessionKey(id), "last_seen", time.Now().Unix()).Err()
	}

	return userID, nil
}

func (s *Store) Delete(ctx context.Context, id string) error {
	rawUserID, err := s.rdb.HGet(ctx, sessionKey(id), "user_id").Result()
	if errors.Is(err, redis.Nil) {
		return nil
	}
	if err != nil {
		return err
	}

	pipe := s.rdb.TxPipeline()
	pipe.Del(ctx, sessionKey(id))
	if userID, convErr := strconv.ParseInt(rawUserID, 10, 64); convErr == nil {
		pipe.SRem(ctx, userKey(userID), id)
	}
	_, err = pipe.Exec(ctx)
	return err
}

func (s *Store) ListForUser(ctx context.Context, userID int64) ([]models.Session, error) {
	ids, err := s.rdb.SMembers(ctx, userKey(userID)).Result()
	if err != nil {
		return nil, err
	}

	list := make([]models.Session, 0, len(ids))
	for _, id := range ids {
		fields, err := s.rdb.HGetAll(ctx, sessionKey(id)).Result()
		if err != nil {
			return nil, err
		}
		if len(fields) == 0 {
			// expired sessions leave dangling set members behind
			_ = s.rdb.SRem(ctx, userKey(userID), id).Err()
			continue
		}

		createdAt, createdErr := strconv.ParseInt(fields["created_at"], 10, 64)
		lastSeen, seenErr := strconv.ParseInt(fields["last_seen"], 10, 64)
		if createdErr != nil || seenErr != nil {
			continue
		}

		list = append(list, models.Session{
			ID:        id,
			UserAgent: fields["user_agent"],
			IP:        fields["ip"],
			CreatedAt: time.Unix(createdAt, 0),
			LastSeen:  time.Unix(lastSeen, 0),
		})
	}

	return list, nil
}

func (s *Store) DeleteAllForUser(ctx context.Context, userID int64) error {
	ids, err := s.rdb.SMembers(ctx, userKey(userID)).Result()
	if err != nil {
		return err
	}

	keys := make([]string, 0, len(ids)+1)
	for _, id := range ids {
		keys = append(keys, sessionKey(id))
	}
	keys = append(keys, userKey(userID))

	return s.rdb.Del(ctx, keys...).Err()
}
