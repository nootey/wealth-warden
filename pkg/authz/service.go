package authz

import (
	"context"
	"gorm.io/gorm"
	"sync"
	"time"
)

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

func (s *Service) LoadPrincipal(ctx context.Context, userID int64) (*Principal, error) {
	type row struct {
		UserID   int64
		RoleID   int64
		RoleName string
	}
	var r row
	if err := s.DB.WithContext(ctx).
		Table("users u").
		Select("u.id as user_id, r.id as role_id, r.name as role_name").
		Joins("JOIN roles r ON r.id = u.role_id").
		Where("u.id = ?", userID).
		First(&r).Error; err != nil {
		return nil, err
	}
	perms := s.getPermsForRole(ctx, r.RoleID)
	return &Principal{UserID: r.UserID, RoleID: r.RoleID, RoleName: r.RoleName, Perms: perms}, nil
}

func (s *Service) getPermsForRole(ctx context.Context, roleID int64) map[string]struct{} {
	s.mu.RLock()
	e, ok := s.roleCache[roleID]
	s.mu.RUnlock()
	if ok && time.Since(e.by) < s.ttl {
		return e.perms
	}
	var names []string
	_ = s.DB.WithContext(ctx).
		Table("permissions p").
		Select("p.name").
		Joins("JOIN role_permissions rp ON rp.permission_id = p.id").
		Where("rp.role_id = ?", roleID).
		Scan(&names)
	set := make(map[string]struct{}, len(names))
	for _, n := range names {
		set[n] = struct{}{}
	}
	s.mu.Lock()
	s.roleCache[roleID] = cachedPerms{perms: set, by: time.Now()}
	s.mu.Unlock()
	return set
}
