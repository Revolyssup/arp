package cache

import (
	"testing"
	"time"

	"github.com/Revolyssup/arp/pkg/logger"
)

func TestLRU(t *testing.T) {
	lru := NewLRUCache[int](2, logger.New(logger.LevelDebug))
	lru.Set("a", 1, time.Second*2)
	lru.Set("b", 2, time.Second*2)
	val, ok := lru.Get("a")
	if !ok || val != 1 {
		t.Errorf("Expected 1, got %+v", val)
	}
	lru.Set("c", 3, time.Second*2) // This should evict "b"
	_, ok = lru.Get("b")
	if ok {
		t.Errorf("Expected b to be evicted")
	}
	time.Sleep(3 * time.Second)
	_, ok = lru.Get("a")
	if ok {
		t.Errorf("Expected a to be expired")
	}
}
