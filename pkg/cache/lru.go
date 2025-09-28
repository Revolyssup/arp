package cache

import (
	"fmt"
	"sync"
	"time"

	"github.com/Revolyssup/arp/pkg/logger"
)

// Note: Since the project is not performance critical and for learning purposes, I am using my custom implementation instead of using hashicorp one.
type LRUCache[T any] struct {
	Head    *Node
	Tail    *Node
	m       map[string]T
	maxSize int
	mx      sync.Mutex
	log     *logger.Logger
}

type Node struct {
	Left  *Node
	Right *Node
	key   string
}

func NewLRUCache[T any](size int, logger *logger.Logger) *LRUCache[T] {
	return &LRUCache[T]{
		m:       make(map[string]T),
		maxSize: size,
		log:     logger.WithComponent("LRUCache"),
	}
}

func (nlru *LRUCache[T]) DebugGet() map[string]T {
	return nlru.m
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

// GET(KEY)-> VALUE. Implement without TTL
// EXPIRE: WHEN THE BUFFER IS FULL.
// Doubly linked list: WHenever a key is accessed, put it at the start of the list.
// Whenever buffer is full, evict from tail
func (lru *LRUCache[T]) pop(key string) *Node {
	var n *Node
	ptr := lru.Head
	for ptr != nil && ptr.key != key {
		ptr = ptr.Right
	}
	if ptr != nil {
		//if there is node on left attach it to right.
		if ptr.Left != nil {
			ptr.Left.Right = ptr.Right
		} else { //popping head
			lru.Head = lru.Head.Right
		}
		if ptr.Right != nil {
			ptr.Right.Left = ptr.Left
		} else { //popping tail
			lru.Tail = lru.Tail.Left
		}
		n = ptr
	}
	return n
}

func (lru *LRUCache[T]) push(node *Node, key string) {
	if node == nil {
		node = &Node{
			key: key,
		}
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
	if node.Right == nil {
		lru.Tail = node
	}
}
func (lru *LRUCache[T]) Get(key string) (ansval T, ok bool) {
	lru.mx.Lock()
	defer lru.mx.Unlock()
	if val, ok := lru.m[key]; ok {
		//put the key at the beginning of list
		node := lru.pop(key)
		lru.push(node, key)
		return val, true
	}
	return ansval, false
}

func (lru *LRUCache[T]) Delete(key string) (ok bool) {
	lru.mx.Lock()
	defer lru.mx.Unlock()
	if _, ok := lru.m[key]; ok {
		delete(lru.m, key)
		lru.pop(key)
		ok = true
	}
	return
}

func (lru *LRUCache[T]) poptail() {
	ptr := lru.Tail
	if ptr.Left != nil {
		ptr.Left.Right = nil
		lru.Tail = ptr.Left
	} else { // no node
		lru.Head = nil
		lru.Tail = nil
	}
}
func (lru *LRUCache[T]) Set(key string, val T, ttl time.Duration) {
	lru.mx.Lock()
	defer lru.mx.Unlock()
	//cleanup
	defer func() {
		if ttl >= 0 {
			go func() {
				<-time.After(ttl)
				lru.Delete(key)
			}()
		}
	}()
	if len(lru.m) == lru.maxSize {
		//evict tail ptr
		delete(lru.m, lru.Tail.key)
		lru.poptail()
	}
	lru.m[key] = val
	//put the key at the beginning of list
	node := lru.pop(key)
	lru.push(node, key)
}
