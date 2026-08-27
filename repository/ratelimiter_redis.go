package repository

import (
	"context"
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

const rateLimitScript = `
local key = KEYS[1]
local now = tonumber(ARGV[1])
local window_ms = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local member = ARGV[4]

redis.call('ZREMRANGEBYSCORE', key, 0, now - window_ms)
local count = redis.call('ZCARD', key)
local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')

local allowed = 0
if count < limit then
    allowed = 1
    redis.call('ZADD', key, now, member)
    redis.call('PEXPIRE', key, window_ms)
end

if oldest[2] then
    return {allowed, limit - count - allowed, oldest[2]}
end
return {allowed, limit - count - allowed, tostring(now)}
`

type rateLimiterRepositoryRedis struct {
	client *redis.Client
	script *redis.Script
	seq    atomic.Int64
}

func NewRateLimiterRepositoryRedis(client *redis.Client) RateLimiterRepository {
	return &rateLimiterRepositoryRedis{client: client, script: redis.NewScript(rateLimitScript)}
}

func (r *rateLimiterRepositoryRedis) Allow(ctx context.Context, key string, limit int, window time.Duration) (RateLimitResult, error) {
	now := time.Now().UnixMilli()
	member := strconv.FormatInt(now, 10) + "-" + strconv.FormatInt(r.seq.Add(1), 10)

	raw, err := r.script.Run(ctx, r.client, []string{rateLimitKey(key)},
		now, window.Milliseconds(), limit, member).Result()
	if err != nil {
		return RateLimitResult{}, fmt.Errorf("rate limit %s: %w", key, err)
	}

	vals, ok := raw.([]interface{})
	if !ok || len(vals) != 3 {
		return RateLimitResult{}, fmt.Errorf("rate limit %s: unexpected script result %v", key, raw)
	}

	allowed, _ := vals[0].(int64)
	remaining, _ := vals[1].(int64)
	oldestMillis, err := strconv.ParseInt(vals[2].(string), 10, 64)
	if err != nil {
		return RateLimitResult{}, fmt.Errorf("rate limit %s: parse reset time: %w", key, err)
	}

	return RateLimitResult{
		Allowed:   allowed == 1,
		Remaining: int(remaining),
		ResetAt:   time.UnixMilli(oldestMillis).Add(window),
	}, nil
}

func rateLimitKey(key string) string {
	return "rate_limit:" + key
}
