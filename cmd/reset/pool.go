package main

import (
	"reflect"
	"sync"
)

type resetter interface {
	Reset()
}

type Pool[T resetter] struct {
	pool sync.Pool
}

func New[T resetter](constructors ...func() T) *Pool[T] {
	var constructor func() T
	if len(constructors) > 0 {
		constructor = constructors[0]
	} else {
		constructor = defaultConstructor[T]
	}

	return &Pool[T]{
		pool: sync.Pool{
			New: func() any {
				return constructor()
			},
		},
	}
}

func (p *Pool[T]) Get() T {
	return p.pool.Get().(T)
}

func (p *Pool[T]) Put(obj T) {
	obj.Reset()
	p.pool.Put(obj)
}

func defaultConstructor[T resetter]() T {
	var zero T
	typ := reflect.TypeOf(zero)
	if typ == nil {
		panic("reset pool: cannot create value for nil interface type")
	}
	if typ.Kind() == reflect.Pointer {
		return reflect.New(typ.Elem()).Interface().(T)
	}

	return reflect.New(typ).Elem().Interface().(T)
}
