package cache

import (
	"container/list"
	"sync"
)

type entry[K comparable, V any] struct {
	key   K
	value V
}

type LRU[K comparable, V any] struct {
	mu    sync.Mutex
	cap   int
	items map[K]*list.Element
	order *list.List
}

func NewLRU[K comparable, V any](cap int) *LRU[K, V] {
	return &LRU[K, V]{cap: cap, items: make(map[K]*list.Element), order: list.New()}
}

func (l *LRU[K, V]) Get(k K) (V, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if el, ok := l.items[k]; ok {
		l.order.MoveToFront(el)
		return el.Value.(entry[K, V]).value, true
	}
	var z V
	return z, false
}

func (l *LRU[K, V]) Set(k K, v V) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if el, ok := l.items[k]; ok {
		el.Value = entry[K, V]{key: k, value: v}
		l.order.MoveToFront(el)
		return
	}
	el := l.order.PushFront(entry[K, V]{key: k, value: v})
	l.items[k] = el
	if l.order.Len() > l.cap {
		last := l.order.Back()
		if last != nil {
			l.order.Remove(last)
			delete(l.items, last.Value.(entry[K, V]).key)
		}
	}
}
