package main

import "container/list"

type entry[K comparable, V any] struct {
	key   K
	value V
}
type LRUCache[K comparable, V any] struct {
	capacity int
	ll       list.List
	items    map[K]*list.Element
}

func NewLRUCache[K comparable, V any](capacity int) *LRUCache[K, V] {
	return &LRUCache[K, V]{capacity: capacity,
		items: make(map[K]*list.Element)}
}
func (c *LRUCache[K, V]) Get(key K) (value V, ok bool) {
	elem, exist := c.items[key]
	if !exist {
		var zero V
		return zero, false
	}
	e := elem.Value.(*entry[K, V]).value
	c.ll.MoveToFront(elem)
	return e, exist
}
func (c *LRUCache[K, V]) Set(key K, value V) {
	if c.capacity <= 0 {
		return
	}
	elem, exist := c.items[key]
	if exist {
		existElem := elem.Value.(*entry[K, V])
		existElem.value = value
		return
	}
	if len(c.items) >= c.capacity {
		oldest := c.ll.Back()
		oldestKey := oldest.Value.(*entry[K, V]).key
		delete(c.items, oldestKey)
		c.ll.Remove(oldest)
	}
	newElem := c.ll.PushFront(&entry[K, V]{key: key, value: value})
	c.items[key] = newElem
}

type LRU[K comparable, V any] interface {
	Get(K) (V, bool)
	Set(K, V)
}
