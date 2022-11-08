package cache

import (
	"errors"
	"sync"
)

var errCacheNotFound = errors.New("lru: 缓存未找到")

type lruCache struct {
	cache      map[string]*dualNode
	head, tail *dualNode
	m          *sync.RWMutex
}

func newLruCache() *lruCache {
	var head, tail *dualNode
	head.next = tail
	tail.prev = head
	return &lruCache{
		cache: nil,
		head:  head,
		tail:  tail,
	}
}

func (l *lruCache) get(key string) (any, error) {
	l.m.Lock()
	defer l.m.Unlock()
	node, ok := l.cache[key]
	if !ok {
		return nil, errCacheNotFound
	}
	node.prev.next = node.next
	node.next.prev = node.prev
	node.next = l.head.next
	l.head.next.prev = node
	l.head.next = node
	node.prev = l.head
	return node.value, nil
}

func (l *lruCache) set(key string, val any) error {
	l.m.Lock()
	defer l.m.Unlock()
	node, ok := l.cache[key]
	if !ok {
		node = &dualNode{key: key}
	}
	node.value = val
	l.cache[key] = node
	node.next = l.head.next
	l.head.next.prev = node
	l.head.next = node
	node.prev = l.head
	return nil
}

func (l *lruCache) getOldest() (string, any, error) {
	l.m.RLock()
	defer l.m.RUnlock()
	node := l.tail.prev
	return node.key, node.value, nil
}

func (l *lruCache) deleteOldest() (string, any, error) {
	l.m.Lock()
	defer l.m.Unlock()
	node := l.tail.prev
	node.prev.next = node.next
	node.next.prev = node.prev
	return node.key, node.value, nil
}

type dualNode struct {
	key        string
	value      any
	prev, next *dualNode
}
