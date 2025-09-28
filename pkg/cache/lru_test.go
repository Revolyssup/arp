package cache

import (
	"fmt"
	"testing"
	"time"

	"github.com/Revolyssup/arp/pkg/logger"
)

func TestLRU(t *testing.T) {
	lru := NewLRUCache[int](2, logger.New(logger.LevelInfo))
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

// BenchmarkLRUCache_Set benchmarks setting values in the cache
func BenchmarkLRUCache_Set(b *testing.B) {
	log := logger.New(logger.LevelInfo).WithComponent("benchmark")
	cache := NewLRUCache[string](1000, log)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i%1000)
		cache.Set(key, "value", -1)
	}
}

// BenchmarkLRUCache_Get benchmarks getting values from the cache
func BenchmarkLRUCache_Get(b *testing.B) {
	log := logger.New(logger.LevelInfo).WithComponent("benchmark")
	cache := NewLRUCache[string](1000, log)

	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key%d", i)
		cache.Set(key, "value", -1)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i%1000)
		cache.Get(key)
	}
}

// BenchmarkLRUCache_GetMiss benchmarks cache misses
func BenchmarkLRUCache_GetMiss(b *testing.B) {
	log := logger.New(logger.LevelInfo).WithComponent("benchmark")
	cache := NewLRUCache[string](1000, log)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("miss%d", i)
		cache.Get(key)
	}
}

// BenchmarkLRUCache_SetGetMixed benchmarks mixed operations
func BenchmarkLRUCache_SetGetMixed(b *testing.B) {
	log := logger.New(logger.LevelInfo).WithComponent("benchmark")
	cache := NewLRUCache[string](1000, log)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i%1000)
		if i%2 == 0 {
			cache.Set(key, "value", -1)
		} else {
			cache.Get(key)
		}
	}
}

// Benchmark different cache sizes
func BenchmarkLRUCache_DifferentSizes(b *testing.B) {
	sizes := []int{10, 100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size_%d", size), func(b *testing.B) {
			log := logger.New(logger.LevelInfo).WithComponent("benchmark")
			cache := NewLRUCache[string](size, log)

			// Pre-populate
			for i := 0; i < size; i++ {
				cache.Set(fmt.Sprintf("key%d", i), "value", -1)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				key := fmt.Sprintf("key%d", i%size)
				cache.Get(key)
			}
		})
	}
}

// Benchmark concurrent access
func BenchmarkLRUCache_Concurrent(b *testing.B) {
	log := logger.New(logger.LevelInfo).WithComponent("benchmark")
	cache := NewLRUCache[string](1000, log)

	// Pre-populate
	for i := 0; i < 1000; i++ {
		cache.Set(fmt.Sprintf("key%d", i), "value", -1)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key%d", i%1000)
			if i%4 == 0 {
				cache.Set(key, "newvalue", -1)
			} else {
				cache.Get(key)
			}
			i++
		}
	})
}
