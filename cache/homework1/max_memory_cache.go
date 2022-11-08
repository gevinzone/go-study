package cache

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type MaxMemoryCache struct {
	Cache
	max         int64
	used        int64
	lruSentinel lru
	m           *sync.RWMutex
}

func NewMaxMemoryCache(max int64, cache Cache) *MaxMemoryCache {
	res := &MaxMemoryCache{
		Cache: cache,
		max:   max,
	}
	res.lruSentinel = newLruCache()
	// 注册回调
	res.OnEvicted(func(key string, val []byte) {
		// 删除key时，修改used 大小
		size := int64(len(val) * 8)
		res.used -= size
	})
	return res
}

func (m *MaxMemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	m.m.RLock()
	defer m.m.RUnlock()
	_, _ = m.lruSentinel.get(key)
	return m.Cache.Get(ctx, key)
}

func (m *MaxMemoryCache) Set(ctx context.Context, key string, val []byte,
	expiration time.Duration) error {
	// 判断内存使用量，超出限制，则通过LRU策略清理内容
	m.m.Lock()
	defer m.m.Unlock()
	size := int64(len(val) * 8)
	if err := m.lruStrategy(ctx, size); err != nil {
		return err
	}

	if err := m.lruSentinel.set(key, ""); err != nil {
		return err
	}
	return m.Cache.Set(ctx, key, val, expiration)
}

func (m *MaxMemoryCache) lruStrategy(ctx context.Context, size int64) error {
	for m.used+size > m.max {
		key, _, err := m.lruSentinel.deleteOldest()
		if err != nil {
			return fmt.Errorf("对象大小超出缓存最大容量, %v", err)
		}
		err = m.Delete(ctx, key)
		if err != nil {
			return err
		}
	}
	return nil
}
