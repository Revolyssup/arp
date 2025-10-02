package utils

import "sync"

type Pool[T any] struct {
	pool *sync.Pool
}

func NewPool[T any](newFunc func() T) *Pool[T] {
	return &Pool[T]{
		pool: &sync.Pool{
			New: func() any {
				return newFunc()
			},
		},
	}
}

func (p *Pool[T]) Get() T {
	item := p.pool.Get()
	if item == nil {
		it := p.pool.New()
		if it == nil {
			var zero T
			return zero
		}
		item = it.(T)
	}
	return item.(T)
}

func (p *Pool[T]) Put(item T) {
	p.pool.Put(item)
}

// shamelessly inspired from traefik

func GoWithRecover(fn func(), recoverFunc func(any)) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				recoverFunc(r)
			}
		}()
		fn()
	}()
}
