package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Revolyssup/arp/pkg/logger"
	"github.com/Revolyssup/arp/pkg/utils"
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

	//for managing the cleanup goroutines for TTL
	ttlCtx     context.Context
	ttlCancel  context.CancelFunc
	keyCancels map[string]context.CancelFunc
	ttlWg      sync.WaitGroup
}

func NewLRUCache[T any](size int, logger *logger.Logger) *LRUCache[T] {
	// Note: Its okay to use Background here as this is the context that will be
	// used for all TTL goroutines. Passing context from NewLRUCache will complicate things for no reason.
	// LRUCache exposes a Destroy() method for cleanup which the caller must respect.
	ctx, cancel := context.WithCancel(context.Background())
	return &LRUCache[T]{
		m:          make(map[string]*Node[T]),
		maxSize:    size,
		log:        logger.WithComponent("LRUCache"),
		ttlCtx:     ctx,
		keyCancels: make(map[string]context.CancelFunc),
		ttlCancel:  cancel,
	}
}

func (nlru *LRUCache[T]) Stop() {
	nlru.log.Debugf("Stopping LRU Cache TTL goroutines")
	nlru.mx.Lock()
	defer nlru.mx.Unlock()

	if nlru.ttlCancel != nil {
		nlru.ttlCancel()
	}
	nlru.ttlWg.Wait()
}

// Resets but keeps the cache operational
func (nlru *LRUCache[T]) Reset() {
	nlru.log.Debugf("Resetting LRU Cache")
	nlru.Stop()

	nlru.mx.Lock()
	defer nlru.mx.Unlock()
	nlru.Head = nil
	nlru.Tail = nil
	nlru.m = make(map[string]*Node[T])

	// recreate context for TTL goroutines
	nlru.ttlCtx, nlru.ttlCancel = context.WithCancel(context.Background())
}

// Destroys completely - cannot be used after this
func (nlru *LRUCache[T]) Destroy() {
	nlru.log.Debugf("Destroying LRU Cache")
	nlru.Stop()
	nlru.mx.Lock()
	defer nlru.mx.Unlock()
	nlru.Head = nil
	nlru.Tail = nil
	nlru.m = nil
	nlru.ttlCtx = nil
	nlru.keyCancels = nil
	nlru.ttlCancel = nil
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

func (lru *LRUCache[T]) cancelKeyTTL(key string) {
	if cancel, exists := lru.keyCancels[key]; exists {
		lru.log.Debugf("Cancelling TTL goroutine for key %s", key)
		cancel()
		delete(lru.keyCancels, key)
	}
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
		// Cancel any existing TTL goroutine for this key
		lru.cancelKeyTTL(key)
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
	// Cancel TTL cleanup go routine for the evicted key
	lru.cancelKeyTTL(lru.Tail.key)

	// Remove from map and list
	delete(lru.m, lru.Tail.key)
	lru.pop(lru.Tail)
}

func (lru *LRUCache[T]) Set(key string, val T, ttl time.Duration) {
	lru.log.Debugf("Setting key %s in LRU Cache with TTL %v", key, ttl)
	lru.mx.Lock()
	defer lru.mx.Unlock()

	// Check if key already exists
	if existingNode, exists := lru.m[key]; exists {
		// Cancel any existing TTL goroutine for this key
		lru.cancelKeyTTL(key)
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

	// TTL handling with proper context
	if ttl >= 0 {
		keyCtx, keyCancel := context.WithCancel(lru.ttlCtx)
		lru.keyCancels[key] = keyCancel
		lru.ttlWg.Add(1)
		utils.GoWithRecover(func() {
			defer lru.ttlWg.Done()

			select {
			case <-keyCtx.Done():
				lru.log.Debugf("TTL context done for key %s", key)
				return
			case <-time.After(ttl):
				lru.log.Debugf("TTL expired for key %s, deleting from LRU Cache after %v", key, ttl)
				lru.Delete(key)
			}
		}, func(err any) {
			lru.log.Infof("panic in LRU cache ttl goroutine: %v", err)
			lru.ttlWg.Done()
		})
	}
}
