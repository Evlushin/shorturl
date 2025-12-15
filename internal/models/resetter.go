package models

import "sync"

// Resettable — интерфейс, ограничивающий типы, которые можно хранить в Pool.
// Требует наличия метода Reset().
type Resettable interface {
	Reset()
}

// Pool — структура-пул для хранения объектов типа T, где T реализует Resettable.
type Pool[T Resettable] struct {
	pool sync.Pool
}

// New — функция-конструктор, создаёт и возвращает указатель на Pool[T].
func New[T Resettable]() *Pool[T] {
	return &Pool[T]{
		pool: sync.Pool{
			New: func() interface{} {
				var zero T
				return zero
			},
		},
	}
}

// Get — метод, возвращающий объект из пула.
// Если в пуле нет свободного объекта, создаётся новый (через New функции sync.Pool).
func (p *Pool[T]) Get() T {
	obj := p.pool.Get().(T)
	return obj
}

// Put — метод, помещающий объект обратно в пул.
// Перед помещением в пул объект сбрасывается (вызывается Reset()).
func (p *Pool[T]) Put(obj T) {
	obj.Reset()
	p.pool.Put(obj)
}
