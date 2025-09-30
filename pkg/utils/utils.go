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
		item = p.pool.New().(T)
	}
	return item.(T)
}

func (p *Pool[T]) Put(item T) {
	p.pool.Put(item)
}
