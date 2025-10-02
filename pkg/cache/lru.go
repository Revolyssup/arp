package cache

import (
	"fmt"
	"sync"
	"time"

	"github.com/Revolyssup/arp/pkg/logger"
)

type Node[T any] struct {
	Left       *Node[T]
	Right      *Node[T]
	key        string
	value      T
	expiration int64 // Unix nano timestamp, 0 means no expiration
}

type LRUCache[T any] struct {
	Head    *Node[T]
	Tail    *Node[T]
	m       map[string]*Node[T]
	maxSize int
	mx      sync.Mutex
	log     *logger.Logger

	// Hashicorp-style TTL management
	janitor     *janitor
	stopJanitor chan struct{}
}

type janitor struct {
	Interval time.Duration
	stop     chan struct{}
}

func NewLRUCache[T any](size int, cleanupInterval time.Duration, logger *logger.Logger) *LRUCache[T] {
	lru := &LRUCache[T]{
		m:           make(map[string]*Node[T]),
		maxSize:     size,
		log:         logger.WithComponent("LRUCache"),
		stopJanitor: make(chan struct{}),
	}

	// Start janitor if cleanup interval is positive
	if cleanupInterval > 0 {
		lru.startJanitor(cleanupInterval)
	}

	return lru
}

func (lru *LRUCache[T]) startJanitor(interval time.Duration) {
	j := &janitor{
		Interval: interval,
		stop:     make(chan struct{}),
	}
	lru.janitor = j

	go lru.runJanitor(j)
}

func (lru *LRUCache[T]) runJanitor(j *janitor) {
	ticker := time.NewTicker(j.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			lru.deleteExpired()
		case <-j.stop:
			return
		case <-lru.stopJanitor:
			return
		}
	}
}

func (lru *LRUCache[T]) deleteExpired() {
	lru.mx.Lock()
	defer lru.mx.Unlock()

	now := time.Now().UnixNano()
	expiredKeys := make([]string, 0)

	// First pass: collect expired keys
	for key, node := range lru.m {
		if node.expiration > 0 && now > node.expiration {
			expiredKeys = append(expiredKeys, key)
		}
	}

	// Second pass: remove expired items
	for _, key := range expiredKeys {
		if node, exists := lru.m[key]; exists {
			lru.log.Debugf("TTL expired for key %s, deleting from LRU Cache", key)
			lru.pop(node)
			delete(lru.m, key)
		}
	}
}

func (lru *LRUCache[T]) Stop() {
	lru.log.Debugf("Stopping LRU Cache janitor")
	lru.mx.Lock()
	defer lru.mx.Unlock()

	// Stop the janitor
	if lru.janitor != nil {
		close(lru.janitor.stop)
		lru.janitor = nil
	}
	close(lru.stopJanitor)
}

func (lru *LRUCache[T]) Reset() {
	lru.log.Debugf("Resetting LRU Cache")
	lru.mx.Lock()
	defer lru.mx.Unlock()

	lru.Head = nil
	lru.Tail = nil
	lru.m = make(map[string]*Node[T])
}

func (lru *LRUCache[T]) Destroy() {
	lru.log.Debugf("Destroying LRU Cache")
	lru.Stop()

	lru.mx.Lock()
	defer lru.mx.Unlock()
	lru.Head = nil
	lru.Tail = nil
	lru.m = nil
}

func (nlru *LRUCache[T]) DebugGet() map[string]T {
	result := make(map[string]T)
	for k, node := range nlru.m {
		result[k] = node.value
	}
	return result
}

func (nlru *LRUCache[T]) PrintList() string {
	tmp := nlru.Head
	ans := ""
	for tmp != nil {
		ans += fmt.Sprintf("%s->", tmp.key)
		tmp = tmp.Right
	}
	return ans
}

// O(1) pop - no traversal needed!
func (lru *LRUCache[T]) pop(node *Node[T]) {
	if node == nil {
		return
	}

	// Remove from linked list
	if node.Left != nil {
		node.Left.Right = node.Right
	} else { // This was the head
		lru.Head = node.Right
	}

	if node.Right != nil {
		node.Right.Left = node.Left
	} else { // This was the tail
		lru.Tail = node.Left
	}

	// Clear pointers
	node.Left = nil
	node.Right = nil
}

func (lru *LRUCache[T]) push(node *Node[T]) {
	if node == nil {
		return
	}

	if lru.Head == nil {
		lru.Head = node
		lru.Tail = node
		return
	}

	node.Left = nil
	node.Right = lru.Head
	lru.Head.Left = node
	lru.Head = node
}

func (lru *LRUCache[T]) Get(key string) (ansval T, ok bool) {
	lru.log.Debugf("Getting key %s from LRU Cache", key)
	lru.mx.Lock()
	defer lru.mx.Unlock()

	if node, ok := lru.m[key]; ok {
		// Check expiration
		if node.expiration > 0 && time.Now().UnixNano() > node.expiration {
			lru.log.Debugf("Key %s found but expired", key)
			lru.pop(node)
			delete(lru.m, key)
			return ansval, false
		}

		// Move to front - O(1) operations
		lru.pop(node)
		lru.push(node)
		return node.value, true
	}
	return ansval, false
}

func (lru *LRUCache[T]) Delete(key string) (ok bool) {
	lru.log.Debugf("Deleting key %s from LRU Cache", key)
	lru.mx.Lock()
	defer lru.mx.Unlock()

	if node, ok := lru.m[key]; ok {
		lru.pop(node)
		delete(lru.m, key)
		return true
	}
	lru.log.Debugf("Key %s not found in LRU Cache for delete", key)
	return false
}

func (lru *LRUCache[T]) poptail() {
	if lru.Tail == nil {
		return
	}
	// Remove from map and list
	delete(lru.m, lru.Tail.key)
	lru.pop(lru.Tail)
}

func (lru *LRUCache[T]) Set(key string, val T, ttl time.Duration) {
	lru.log.Debugf("Setting key %s in LRU Cache with TTL %v", key, ttl)
	lru.mx.Lock()
	defer lru.mx.Unlock()

	var expiration int64
	if ttl > 0 {
		expiration = time.Now().Add(ttl).UnixNano()
	}

	// Check if key already exists
	if existingNode, exists := lru.m[key]; exists {
		// Update value, expiration and move to front
		existingNode.value = val
		existingNode.expiration = expiration
		lru.pop(existingNode)
		lru.push(existingNode)
	} else {
		// Create new node
		newNode := &Node[T]{
			key:        key,
			value:      val,
			expiration: expiration,
		}

		// Evict if needed
		if len(lru.m) >= lru.maxSize {
			lru.poptail()
		}

		// Add to map and list
		lru.m[key] = newNode
		lru.push(newNode)
	}
}
