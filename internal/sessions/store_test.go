package sessions_test

import (
	"context"
	"testing"
	"time"
	"wealth-warden/internal/sessions"
	"wealth-warden/pkg/config"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) (*sessions.Store, *miniredis.Miniredis, *redis.Client) {
	t.Helper()

	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	store := sessions.NewStore(client, config.SessionConfig{
		TTLHours:           24,
		RememberMeTTLHours: 720,
		MaxLifetimeHours:   2160,
	})
	return store, mr, client
}

func isMember(t *testing.T, mr *miniredis.Miniredis, key, member string) bool {
	t.Helper()
	ok, err := mr.SIsMember(key, member)
	if err != nil {
		return false
	}
	return ok
}

func TestCreateAndValidate(t *testing.T) {
	store, mr, _ := newTestStore(t)
	ctx := context.Background()

	id, err := store.Create(ctx, 42, false, "test-agent", "127.0.0.1")
	require.NoError(t, err)
	require.NotEmpty(t, id)

	userID, err := store.Validate(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, int64(42), userID)

	assert.Equal(t, 24*time.Hour, mr.TTL("session:"+id))
	assert.True(t, isMember(t, mr, "user_sessions:42", id))
}

func TestCreateRememberMeUsesLongTTL(t *testing.T) {
	store, mr, _ := newTestStore(t)

	id, err := store.Create(context.Background(), 42, true, "", "")
	require.NoError(t, err)

	assert.Equal(t, 720*time.Hour, mr.TTL("session:"+id))
}

func TestValidateUnknownID(t *testing.T) {
	store, _, _ := newTestStore(t)

	_, err := store.Validate(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, sessions.ErrNotFound)
}

func TestValidateExpiredSession(t *testing.T) {
	store, mr, _ := newTestStore(t)
	ctx := context.Background()

	id, err := store.Create(ctx, 42, false, "", "")
	require.NoError(t, err)

	mr.FastForward(24*time.Hour + time.Second)

	_, err = store.Validate(ctx, id)
	assert.ErrorIs(t, err, sessions.ErrNotFound)
}

func TestValidateMaxLifetimeExceeded(t *testing.T) {
	store, mr, client := newTestStore(t)
	ctx := context.Background()

	id, err := store.Create(ctx, 42, true, "", "")
	require.NoError(t, err)

	tooOld := time.Now().Add(-2161 * time.Hour).Unix()
	require.NoError(t, client.HSet(ctx, "session:"+id, "created_at", tooOld).Err())

	_, err = store.Validate(ctx, id)
	assert.ErrorIs(t, err, sessions.ErrNotFound)

	assert.False(t, mr.Exists("session:"+id))
	assert.False(t, isMember(t, mr, "user_sessions:42", id))
}

func TestValidateBumpsLastSeenWithoutExtendingExpiry(t *testing.T) {
	store, mr, client := newTestStore(t)
	ctx := context.Background()

	id, err := store.Create(ctx, 42, false, "", "")
	require.NoError(t, err)

	stale := time.Now().Add(-2 * time.Hour).Unix()
	require.NoError(t, client.HSet(ctx, "session:"+id, "last_seen", stale).Err())
	mr.SetTTL("session:"+id, time.Minute)

	_, err = store.Validate(ctx, id)
	require.NoError(t, err)

	assert.Equal(t, time.Minute, mr.TTL("session:"+id), "expiry is fixed at login and must not slide")
	lastSeen, err := client.HGet(ctx, "session:"+id, "last_seen").Int64()
	require.NoError(t, err)
	assert.Greater(t, lastSeen, stale)
}

func TestValidateSkipsLastSeenBumpWithinThrottle(t *testing.T) {
	store, _, client := newTestStore(t)
	ctx := context.Background()

	id, err := store.Create(ctx, 42, false, "", "")
	require.NoError(t, err)
	before, err := client.HGet(ctx, "session:"+id, "last_seen").Int64()
	require.NoError(t, err)

	_, err = store.Validate(ctx, id)
	require.NoError(t, err)

	after, err := client.HGet(ctx, "session:"+id, "last_seen").Int64()
	require.NoError(t, err)
	assert.Equal(t, before, after)
}

func TestDelete(t *testing.T) {
	store, mr, _ := newTestStore(t)
	ctx := context.Background()

	id, err := store.Create(ctx, 42, false, "", "")
	require.NoError(t, err)

	require.NoError(t, store.Delete(ctx, id))

	assert.False(t, mr.Exists("session:"+id))
	assert.False(t, isMember(t, mr, "user_sessions:42", id))

	require.NoError(t, store.Delete(ctx, id), "deleting a gone session is a no-op")
}

func TestDeleteAllForUser(t *testing.T) {
	store, mr, _ := newTestStore(t)
	ctx := context.Background()

	id1, err := store.Create(ctx, 42, false, "", "")
	require.NoError(t, err)
	id2, err := store.Create(ctx, 42, true, "", "")
	require.NoError(t, err)
	other, err := store.Create(ctx, 7, false, "", "")
	require.NoError(t, err)

	require.NoError(t, store.DeleteAllForUser(ctx, 42))

	assert.False(t, mr.Exists("session:"+id1))
	assert.False(t, mr.Exists("session:"+id2))
	assert.False(t, mr.Exists("user_sessions:42"))
	assert.True(t, mr.Exists("session:"+other))
}
