package cache

import (
	"fmt"
	"sync"
	"time"

	"github.com/Revolyssup/arp/pkg/logger"
)

type Node[T any] struct {
	Left  *Node[T]
	Right *Node[T]
	key   string
	value T
}

type LRUCache[T any] struct {
	Head    *Node[T]
	Tail    *Node[T]
	m       map[string]*Node[T] // ← Store NODES, not just values
	maxSize int
	mx      sync.Mutex
	log     *logger.Logger
}

func NewLRUCache[T any](size int, logger *logger.Logger) *LRUCache[T] {
	return &LRUCache[T]{
		m:       make(map[string]*Node[T]),
		maxSize: size,
		log:     logger.WithComponent("LRUCache"),
	}
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
	lru.log.Debugf("Setting key %s in LRU Cache", key)
	lru.mx.Lock()
	defer lru.mx.Unlock()

	// Check if key already exists
	if existingNode, exists := lru.m[key]; exists {
		// Update value and move to front
		existingNode.value = val
		lru.pop(existingNode)
		lru.push(existingNode)
	} else {
		// Create new node
		newNode := &Node[T]{
			key:   key,
			value: val,
		}

		// Evict if needed
		if len(lru.m) >= lru.maxSize {
			lru.poptail()
		}

		// Add to map and list
		lru.m[key] = newNode
		lru.push(newNode)
	}

	// TTL handling (same as before)
	if ttl >= 0 {
		go func(key string) {
			<-time.After(ttl)
			lru.log.Debugf("TTL expired for key %s, deleting from LRU Cache after %v", key, ttl)
			lru.Delete(key)
		}(key)
	}
}
