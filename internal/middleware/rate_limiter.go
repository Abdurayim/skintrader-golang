package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	rdb *redis.Client
}

func NewRateLimiter(rdb *redis.Client) *RateLimiter {
	return &RateLimiter{rdb: rdb}
}

type RateLimitConfig struct {
	Max    int
	Window time.Duration
	Prefix string
}

var (
	StrictLimit  = RateLimitConfig{Max: 10, Window: 15 * time.Minute, Prefix: "rl:strict"}
	APILimit     = RateLimitConfig{Max: 100, Window: 15 * time.Minute, Prefix: "rl:api"}
	SearchLimit  = RateLimitConfig{Max: 30, Window: 1 * time.Minute, Prefix: "rl:search"}
	UploadLimit  = RateLimitConfig{Max: 50, Window: 1 * time.Hour, Prefix: "rl:upload"}
	MessageLimit = RateLimitConfig{Max: 20, Window: 1 * time.Minute, Prefix: "rl:msg"}
	AdminLimit   = RateLimitConfig{Max: 500, Window: 15 * time.Minute, Prefix: "rl:admin"}
)

func (rl *RateLimiter) Limit(cfg RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.Background()
		ip := c.ClientIP()
		key := fmt.Sprintf("%s:%s", cfg.Prefix, ip)

		// Use Redis MULTI for atomic increment + TTL
		pipe := rl.rdb.Pipeline()
		incrCmd := pipe.Incr(ctx, key)
		pipe.Expire(ctx, key, cfg.Window)
		_, err := pipe.Exec(ctx)
		if err != nil {
			// Allow request if Redis is down
			c.Next()
			return
		}

		count := incrCmd.Val()
		remaining := cfg.Max - int(count)
		if remaining < 0 {
			remaining = 0
		}

		resetAt := time.Now().Add(cfg.Window).Unix()

		c.Header("X-RateLimit-Limit", strconv.Itoa(cfg.Max))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(resetAt, 10))

		if int(count) > cfg.Max {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"message": "Too many requests, please try again later",
				"code":    "TOO_MANY_REQUESTS",
			})
			return
		}

		c.Next()
	}
}
