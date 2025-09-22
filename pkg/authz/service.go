package authz

import (
	"context"
	"errors"
	"sync"
	"time"

	"gorm.io/gorm"
)

const CtxPermsKey = "perms"

type Service struct {
	DB        *gorm.DB
	mu        sync.RWMutex
	roleCache map[int64]cachedPerms // roleID -> perms
	ttl       time.Duration
}
type cachedPerms struct {
	perms map[string]struct{}
	by    time.Time
}

func NewService(db *gorm.DB, ttl time.Duration) *Service {
	return &Service{DB: db, ttl: ttl, roleCache: make(map[int64]cachedPerms)}
}

func (s *Service) PermsForUser(ctx context.Context, userID int64) (map[string]struct{}, error) {
	// find role for the user
	var row struct {
		RoleID int64
	}
	if err := s.DB.WithContext(ctx).
		Table("users AS u").
		Select("u.role_id AS role_id").
		Where("u.id = ?", userID).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	return s.permsForRole(ctx, row.RoleID)
}

func (s *Service) permsForRole(ctx context.Context, roleID int64) (map[string]struct{}, error) {
	// serve from cache
	s.mu.RLock()
	if e, ok := s.roleCache[roleID]; ok && time.Since(e.by) < s.ttl {
		s.mu.RUnlock()
		return e.perms, nil
	}
	s.mu.RUnlock()

	// load from DB
	var names []string
	if err := s.DB.WithContext(ctx).
		Table("permissions AS p").
		Select("p.name").
		Joins("JOIN role_permissions rp ON rp.permission_id = p.id").
		Where("rp.role_id = ?", roleID).
		Scan(&names).Error; err != nil {
		return nil, err
	}

	set := make(map[string]struct{}, len(names))
	for _, n := range names {
		set[n] = struct{}{}
	}

	// cache
	s.mu.Lock()
	s.roleCache[roleID] = cachedPerms{perms: set, by: time.Now()}
	s.mu.Unlock()
	return set, nil
}
