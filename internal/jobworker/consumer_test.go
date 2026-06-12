package jobworker

import (
	"testing"
	"time"
)

// TestRetryBackoff pins the retry semantics: 1m after the first failure, 2m
// after each subsequent one, dead-letter once attempts reaches maxAttempts.
func TestRetryBackoff(t *testing.T) {
	c := &Consumer{
		maxAttempts:       5,
		initialBackoff:    time.Minute,
		subsequentBackoff: 2 * time.Minute,
	}

	cases := []struct {
		attempts     int
		wantBackoff  time.Duration
		wantDeadLett bool
	}{
		{attempts: 1, wantBackoff: time.Minute, wantDeadLett: false},
		{attempts: 2, wantBackoff: 2 * time.Minute, wantDeadLett: false},
		{attempts: 4, wantBackoff: 2 * time.Minute, wantDeadLett: false},
		{attempts: 5, wantBackoff: 0, wantDeadLett: true},
		{attempts: 6, wantBackoff: 0, wantDeadLett: true},
	}

	for _, tc := range cases {
		backoff, dead := c.retryBackoff(tc.attempts)
		if backoff != tc.wantBackoff || dead != tc.wantDeadLett {
			t.Errorf("retryBackoff(%d) = (%v, %v), want (%v, %v)", tc.attempts, backoff, dead, tc.wantBackoff, tc.wantDeadLett)
		}
	}
}
