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

// Fixed-window counter with an exponential block. KEYS: count, block, strikes.
// ARGV: max, window ms, max block ms, strikes TTL ms. Returns {allowed (1/0), retry after ms}.
// Each time the limit is passed, the strike count goes up and the block lasts
// window * 2^(strikes-1), capped at max block. Strikes are forgotten after strikes TTL.
var backoffScript = redis.NewScript(`
local blocked = redis.call('PTTL', KEYS[2])
if blocked > 0 then
	return {0, blocked}
end
local n = redis.call('INCR', KEYS[1])
if n == 1 then
	redis.call('PEXPIRE', KEYS[1], ARGV[2])
end
if n <= tonumber(ARGV[1]) then
	return {1, 0}
end
local strikes = redis.call('INCR', KEYS[3])
redis.call('PEXPIRE', KEYS[3], ARGV[4])
local block = tonumber(ARGV[2]) * math.pow(2, math.min(strikes - 1, 30))
block = math.floor(math.min(block, tonumber(ARGV[3])))
redis.call('SET', KEYS[2], '1', 'PX', block)
redis.call('DEL', KEYS[1])
return {0, block}
`)

// HitBackoff returns a hit function like Hit, but each block is twice as long as the one before
// (window, 2×window, 4×window, ...) up to maxBlock. The doubling resets after strikesTTL without a block.
func HitBackoff(maxBlock, strikesTTL time.Duration) func(ctx context.Context, key string, max int64, window time.Duration) (bool, time.Duration, error) {
	return func(ctx context.Context, key string, max int64, window time.Duration) (bool, time.Duration, error) {
		// The hash tag keeps the three keys in one slot on Redis Cluster.
		tag := "{" + key + "}"
		keys := []string{tag + ":count", tag + ":block", tag + ":strikes"}
		res, err := backoffScript.Run(ctx, RedisClient, keys,
			max, window.Milliseconds(), maxBlock.Milliseconds(), strikesTTL.Milliseconds()).Int64Slice()
		if err != nil {
			return false, 0, err
		}
		return res[0] == 1, time.Duration(res[1]) * time.Millisecond, nil
	}
}
