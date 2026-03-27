package httpserver

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/cache"
	perrors "github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/errors"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/response"
)

type requestRateLimiter interface {
	allow(c *gin.Context, key string) (bool, int, error)
}

type inMemoryRateLimiter struct {
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

func newInMemoryRateLimiter(rate, burst int) *inMemoryRateLimiter {
	rl := &inMemoryRateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate,
		burst:    burst,
		interval: time.Second,
	}

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

func (rl *inMemoryRateLimiter) allow(_ *gin.Context, key string) (bool, int, error) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[key]
	if !exists {
		rl.visitors[key] = &visitor{tokens: rl.burst - 1, lastSeen: time.Now()}
		return true, rl.burst - 1, nil
	}

	elapsed := time.Since(v.lastSeen)
	refill := int(elapsed/rl.interval) * rl.rate
	if refill > 0 {
		v.tokens += refill
		if v.tokens > rl.burst {
			v.tokens = rl.burst
		}
		v.lastSeen = time.Now()
	}

	if v.tokens <= 0 {
		return false, 0, nil
	}

	v.tokens--
	return true, v.tokens, nil
}

type redisRateLimiter struct {
	client       redis.UniversalClient
	rate         int64
	burst        int64
	intervalMS   int64
	keyTTLMillis int64
}

var redisTokenBucketScript = redis.NewScript(`
local rate = tonumber(ARGV[1])
local burst = tonumber(ARGV[2])
local interval = tonumber(ARGV[3])
local now = tonumber(ARGV[4])
local ttl = tonumber(ARGV[5])

local values = redis.call('HMGET', KEYS[1], 'tokens', 'last')
local tokens = tonumber(values[1])
local last = tonumber(values[2])

if tokens == nil then
  tokens = burst
  last = now
else
  local elapsed = now - last
  if elapsed > 0 then
    local refill = math.floor(elapsed / interval) * rate
    if refill > 0 then
      tokens = math.min(burst, tokens + refill)
      last = now
    end
  end
end

local allowed = 0
if tokens > 0 then
  tokens = tokens - 1
  allowed = 1
end

redis.call('HMSET', KEYS[1], 'tokens', tokens, 'last', last)
redis.call('PEXPIRE', KEYS[1], ttl)

return {allowed, tokens}
`)

func newRedisRateLimiter(client redis.UniversalClient, rate, burst int) *redisRateLimiter {
	ttl := 3 * time.Minute
	if rate > 0 && burst > 0 {
		secondsToRefill := (burst / rate) + 1
		if secondsToRefill > 0 {
			ttl = time.Duration(secondsToRefill*3) * time.Second
		}
	}
	if ttl < 30*time.Second {
		ttl = 30 * time.Second
	}

	return &redisRateLimiter{
		client:       client,
		rate:         int64(rate),
		burst:        int64(burst),
		intervalMS:   int64(time.Second / time.Millisecond),
		keyTTLMillis: int64(ttl / time.Millisecond),
	}
}

func (rl *redisRateLimiter) allow(c *gin.Context, key string) (bool, int, error) {
	if rl == nil || rl.client == nil {
		return false, 0, fmt.Errorf("redis limiter is not configured")
	}

	now := time.Now().UnixMilli()
	redisKey := "ratelimit:" + key
	result, err := redisTokenBucketScript.Run(
		c.Request.Context(),
		rl.client,
		[]string{redisKey},
		rl.rate,
		rl.burst,
		rl.intervalMS,
		now,
		rl.keyTTLMillis,
	).Result()
	if err != nil {
		return false, 0, err
	}

	values, ok := result.([]interface{})
	if !ok || len(values) != 2 {
		return false, 0, fmt.Errorf("unexpected redis limiter response")
	}

	allowedInt, err := redisResultInt(values[0])
	if err != nil {
		return false, 0, err
	}
	remaining, err := redisResultInt(values[1])
	if err != nil {
		return false, 0, err
	}
	if remaining < 0 {
		remaining = 0
	}

	return allowedInt == 1, remaining, nil
}

func redisResultInt(v interface{}) (int, error) {
	switch value := v.(type) {
	case int64:
		return int(value), nil
	case int:
		return value, nil
	case string:
		n, err := strconv.Atoi(value)
		if err != nil {
			return 0, err
		}
		return n, nil
	default:
		return 0, fmt.Errorf("unsupported redis result type %T", v)
	}
}

func rateLimitMiddleware(requestsPerSecond, burst int) gin.HandlerFunc {
	fallbackLimiter := newInMemoryRateLimiter(requestsPerSecond, burst)
	limiter := requestRateLimiter(fallbackLimiter)
	if cache.DistributedModeEnabled() {
		if redisClient := cache.GetRedisClient(); redisClient != nil {
			limiter = newRedisRateLimiter(redisClient, requestsPerSecond, burst)
		}
	}

	return func(c *gin.Context) {
		key := rateLimitKey(c)
		allowed, remaining, err := limiter.allow(c, key)
		if err != nil {
			allowed, remaining, _ = fallbackLimiter.allow(c, key)
		}

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

func rateLimitKey(c *gin.Context) string {
	if userID := strings.TrimSpace(c.GetString("userID")); userID != "" {
		endpoint := c.FullPath()
		if endpoint == "" {
			endpoint = c.Request.URL.Path
		}
		return "user:" + userID + "|endpoint:" + endpoint
	}

	clientIP := strings.TrimSpace(c.ClientIP())
	if clientIP == "" {
		clientIP = "unknown"
	}
	return "ip:" + clientIP
}
