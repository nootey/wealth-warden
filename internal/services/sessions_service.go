package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"wealth-warden/internal/models"
	"wealth-warden/internal/sessions"
	"wealth-warden/internal/ws"
	"wealth-warden/pkg/utils"
)

var ErrCannotRevokeCurrentSession = errors.New("log out to end the current session")

type SessionsServiceInterface interface {
	ListSessions(ctx context.Context, userID int64, currentSessionID string) ([]models.SessionInfo, error)
	RevokeSession(ctx context.Context, userID int64, currentSessionID, handle string) error
	RevokeAllSessions(ctx context.Context, userID int64) error
}

type SessionsService struct {
	store *sessions.Store
	hub   *ws.Hub
}

func NewSessionsService(store *sessions.Store, hub *ws.Hub) *SessionsService {
	return &SessionsService{store: store, hub: hub}
}

var _ SessionsServiceInterface = (*SessionsService)(nil)

// The raw session ID never leaves the server; rows are addressed by its hash.
func sessionHandle(id string) string {
	sum := sha256.Sum256([]byte(id))
	return hex.EncodeToString(sum[:])
}

func (s *SessionsService) ListSessions(ctx context.Context, userID int64, currentSessionID string) ([]models.SessionInfo, error) {
	list, err := s.store.ListForUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	resp := make([]models.SessionInfo, 0, len(list))
	for _, sess := range list {
		resp = append(resp, models.SessionInfo{
			ID:        sessionHandle(sess.ID),
			Device:    utils.DeviceFromUserAgent(sess.UserAgent),
			IP:        sess.IP,
			CreatedAt: sess.CreatedAt,
			LastSeen:  sess.LastSeen,
			Current:   sess.ID == currentSessionID,
		})
	}
	sort.Slice(resp, func(i, j int) bool {
		if resp[i].Current != resp[j].Current {
			return resp[i].Current
		}
		return resp[i].LastSeen.After(resp[j].LastSeen)
	})

	return resp, nil
}

func (s *SessionsService) RevokeSession(ctx context.Context, userID int64, currentSessionID, handle string) error {
	list, err := s.store.ListForUser(ctx, userID)
	if err != nil {
		return err
	}

	for _, sess := range list {
		if sessionHandle(sess.ID) != handle {
			continue
		}
		if sess.ID == currentSessionID {
			return ErrCannotRevokeCurrentSession
		}
		if err := s.store.Delete(ctx, sess.ID); err != nil {
			return err
		}
		s.hub.CloseSession(userID, sess.ID)
		return nil
	}

	return sessions.ErrNotFound
}

func (s *SessionsService) RevokeAllSessions(ctx context.Context, userID int64) error {
	if err := s.store.DeleteAllForUser(ctx, userID); err != nil {
		return err
	}
	s.hub.CloseUser(userID)
	return nil
}
