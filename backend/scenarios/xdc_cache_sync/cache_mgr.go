package xdccachesync

import (
	"sync"

	"github.com/go-redis/redis/v8"
)

type LocalCache struct {
	data map[string]interface{}
	mu   sync.RWMutex
}

type CacheManager struct {
	redisClient *redis.Client
	localCache  *LocalCache
	stats       CacheStats
	mu          sync.RWMutex
}

func NewLocalCache() *LocalCache {
	return &LocalCache{
		data: make(map[string]interface{}),
	}
}

func NewCacheManager(redisClient *redis.Client, localCache *LocalCache) *CacheManager {
	return &CacheManager{
		redisClient: redisClient,
		localCache:  localCache,
		stats:       CacheStats{},
	}
}

func (lc *LocalCache) Get(key string) (interface{}, bool) {
	lc.mu.RLock()
	defer lc.mu.RUnlock()
	val, ok := lc.data[key]
	return val, ok
}

func (lc *LocalCache) Set(key string, value interface{}) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.data[key] = value
}

func (lc *LocalCache) Delete(key string) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	delete(lc.data, key)
}

func (lc *LocalCache) Keys() []string {
	lc.mu.RLock()
	defer lc.mu.RUnlock()
	keys := make([]string, 0, len(lc.data))
	for k := range lc.data {
		keys = append(keys, k)
	}
	return keys
}

func (cm *CacheManager) ResetStats() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.stats = CacheStats{}
}

func (cm *CacheManager) IncrementLocalHit() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.stats.LocalHits++
}

func (cm *CacheManager) IncrementLocalMiss() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.stats.LocalMisses++
}

func (cm *CacheManager) IncrementRedisHit() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.stats.RedisHits++
}

func (cm *CacheManager) IncrementRedisMiss() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.stats.RedisMisses++
}

func (cm *CacheManager) IncrementDBQuery() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.stats.DBQueries++
}

func (cm *CacheManager) GetStats() CacheStats {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.stats
}
