package cache

import (
	"context"
	"cboard/v2/internal/database"
)

// ClearAllSubscriptionCache globally flushes all generated subscription payload caches.
// Call this when global nodes, custom nodes, or node scraper updates the node list.
func ClearAllSubscriptionCache() {
	r := database.GetRedis()
	if r == nil {
		return
	}
	ctx := context.Background()
	var cursor uint64
	for {
		keys, nextCursor, err := r.Scan(ctx, cursor, "sub_payload:*", 100).Result()
		if err != nil {
			return
		}
		if len(keys) > 0 {
			r.Del(ctx, keys...)
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
}
