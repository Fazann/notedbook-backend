package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Counts one attempt; sets the TTL on the first attempt and again on the first one over the limit,
// so the block lasts the full duration from that point. Returns {count, ttl in ms}.
var hitScript = redis.NewScript(`
local n = redis.call('INCR', KEYS[1])
if n == 1 or n == tonumber(ARGV[1]) + 1 then
	redis.call('PEXPIRE', KEYS[1], ARGV[2])
end
return {n, redis.call('PTTL', KEYS[1])}
`)

// Hit records one attempt on key within window. It returns false once more than max attempts were
// made; the key then stays blocked for window, counted from the first attempt over the limit.
// retryAfter is the time left until the key resets.
func Hit(ctx context.Context, key string, max int64, window time.Duration) (allowed bool, retryAfter time.Duration, err error) {
	res, err := hitScript.Run(ctx, RedisClient, []string{key}, max, window.Milliseconds()).Int64Slice()
	if err != nil {
		return false, 0, err
	}
	return res[0] <= max, time.Duration(res[1]) * time.Millisecond, nil
}

// Reset clears the attempts counted on key, e.g. after a successful login.
func Reset(ctx context.Context, key string) error {
	return RedisClient.Del(ctx, key).Err()
}
