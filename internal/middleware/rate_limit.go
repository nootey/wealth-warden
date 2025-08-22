package middleware

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
	"sync"
)

type RateLimiter struct {
	mu    sync.Mutex
	m     map[string]*rate.Limiter
	limit rate.Limit
	burst int
}

func NewRateLimiter(rps float64, burst int) *RateLimiter {
	return &RateLimiter{
		m:     make(map[string]*rate.Limiter),
		limit: rate.Limit(rps),
		burst: burst,
	}
}

func (rl *RateLimiter) get(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	l, ok := rl.m[key]
	if !ok {
		l = rate.NewLimiter(rl.limit, rl.burst)
		rl.m[key] = l
	}
	return l
}

// Middleware limits by userID if set by your auth middleware; otherwise by client IP.
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := clientKey(c)
		lim := rl.get(key)

		if !lim.Allow() {
			// hint the client to back off a little
			c.Header("Retry-After", "1") // seconds
			c.AbortWithStatusJSON(429, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}

func clientKey(c *gin.Context) string {
	if v, ok := c.Get("userID"); ok && v != nil {
		return fmt.Sprintf("u:%v", v)
	}
	// fall back to IP for unauthenticated traffic
	return "ip:" + c.ClientIP()
}
