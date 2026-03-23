package httpserver

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	perrors "github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/errors"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/response"
)

type rateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rate     int           // tokens added per interval
	burst    int           // max tokens
	interval time.Duration // refill interval
}

type visitor struct {
	tokens   int
	lastSeen time.Time
}

func newRateLimiter(rate, burst int) *rateLimiter {
	rl := &rateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate,
		burst:    burst,
		interval: time.Second,
	}

	// cleanup goroutine
	go func() {
		for {
			time.Sleep(time.Minute)
			rl.mu.Lock()
			for ip, v := range rl.visitors {
				if time.Since(v.lastSeen) > 3*time.Minute {
					delete(rl.visitors, ip)
				}
			}
			rl.mu.Unlock()
		}
	}()

	return rl
}

func (rl *rateLimiter) allow(key string) (bool, int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[key]
	if !exists {
		rl.visitors[key] = &visitor{tokens: rl.burst - 1, lastSeen: time.Now()}
		return true, rl.burst - 1
	}

	elapsed := time.Since(v.lastSeen)
	refill := int(elapsed / rl.interval) * rl.rate
	if refill > 0 {
		v.tokens += refill
		if v.tokens > rl.burst {
			v.tokens = rl.burst
		}
		v.lastSeen = time.Now()
	}

	if v.tokens <= 0 {
		return false, 0
	}

	v.tokens--
	return true, v.tokens
}

func rateLimitMiddleware(requestsPerSecond, burst int) gin.HandlerFunc {
	limiter := newRateLimiter(requestsPerSecond, burst)

	return func(c *gin.Context) {
		key := c.ClientIP()
		allowed, remaining := limiter.allow(key)

		c.Writer.Header().Set("X-RateLimit-Limit", strconv.Itoa(burst))
		c.Writer.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))

		if !allowed {
			c.Writer.Header().Set("Retry-After", "1")
			response.Error(c, http.StatusTooManyRequests, perrors.CodeRateLimitExceeded, "rate limit exceeded", nil)
			c.Abort()
			return
		}

		c.Next()
	}
}
