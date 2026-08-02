// Package ratelimit provides a Redis-backed sliding-window rate limiter.
package ratelimit

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config holds rate-limiter settings.
type Config struct {
	RequestsPerMinute int
	Burst             int
	Window            time.Duration
	SkipPaths         []string
}

// Limiter is a Redis-backed sliding-window rate limiter.
type Limiter struct {
	rdb    *redis.Client
	cfg    Config
	script *redis.Script
}

// slidingWindowLua implements a token-bucket style sliding window in Lua so
// the check-and-decrement is atomic.
const slidingWindowLua = `
local key        = KEYS[1]
local now        = tonumber(ARGV[1])
local window_ms  = tonumber(ARGV[2])
local max_reqs   = tonumber(ARGV[3])

-- Remove requests outside the current window.
redis.call("ZREMRANGEBYSCORE", key, 0, now - window_ms)

local count = redis.call("ZCARD", key)
if count < max_reqs then
    -- Add current request timestamp with a unique member (now + random suffix).
    redis.call("ZADD", key, now, now .. "-" .. math.random(1, 1000000))
    redis.call("PEXPIRE", key, window_ms)
    return {1, max_reqs - count - 1}   -- allowed, remaining
else
    return {0, 0}                       -- rejected, remaining=0
end
`

// New creates a Limiter connected to the given Redis client.
func New(rdb *redis.Client, cfg Config) *Limiter {
	if cfg.Window == 0 {
		cfg.Window = time.Minute
	}
	if cfg.RequestsPerMinute == 0 {
		cfg.RequestsPerMinute = 60
	}
	return &Limiter{
		rdb:    rdb,
		cfg:    cfg,
		script: redis.NewScript(slidingWindowLua),
	}
}

// Result is the outcome of a rate-limit check.
type Result struct {
	Allowed   bool
	Remaining int
	ResetAt   time.Time
}

// Allow checks whether the given key is within the rate limit.
// key should uniquely identify the caller (e.g. "ratelimit:<ip>:<route>").
func (l *Limiter) Allow(ctx context.Context, key string) (Result, error) {
	nowMs := time.Now().UnixMilli()
	windowMs := l.cfg.Window.Milliseconds()
	maxReqs := l.cfg.RequestsPerMinute

	res, err := l.script.Run(ctx, l.rdb,
		[]string{key},
		strconv.FormatInt(nowMs, 10),
		strconv.FormatInt(windowMs, 10),
		strconv.Itoa(maxReqs),
	).Int64Slice()
	if err != nil {
		// On Redis error, fail open (allow the request).
		return Result{Allowed: true, Remaining: maxReqs}, fmt.Errorf("rate limit script: %w", err)
	}

	allowed := res[0] == 1
	remaining := int(res[1])
	resetAt := time.Now().Add(l.cfg.Window)

	return Result{
		Allowed:   allowed,
		Remaining: remaining,
		ResetAt:   resetAt,
	}, nil
}

// KeyForRequest builds a rate-limit key from IP + path prefix.
func KeyForRequest(ip, path string) string {
	return fmt.Sprintf("ratelimit:%s:%s", ip, path)
}
