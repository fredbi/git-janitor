package registry

import (
	"fmt"
	"iter"
)

type Registrable interface {
	Name() string
}

// Registry holds all registered checks or actions, keyed by name.
//
// It enforces uniqueness and provides iteration for config/wizard discovery.
type Registry[T Registrable] struct {
	index map[string]int
	items []T
}

// New creates an empty Registry.
func New[T Registrable](opts ...Option[T]) *Registry[T] {
	var o options[T]
	for _, apply := range opts {
		apply(&o)
	}

	r := &Registry[T]{
		index: make(map[string]int),
	}

	for _, producer := range o.producers {
		for check := range producer {
			r.register(check)
		}
	}

	return r
}

// register adds a check to the registry. Panics if the name is already taken.
func (r *Registry[T]) register(c T) {
	name := c.Name()
	if _, exists := r.index[name]; exists {
		panic(fmt.Errorf("check %q already registered: %w", name, ErrDuplicate))
	}

	r.index[name] = len(r.items)
	r.items = append(r.items, c)
}

// Get returns a check by name.
//
//nolint:ireturn // returns an interface by design
func (r *Registry[T]) Get(name string) (T, bool) {
	var zero T
	i, ok := r.index[name]
	if !ok {
		return zero, false
	}

	return r.items[i], true
}

// Len returns the number of registered checks.
func (r *Registry[T]) Len() int {
	return len(r.items)
}

// All iterates over all registered checks in insertion order.
func (r *Registry[T]) All() iter.Seq2[string, T] {
	return func(yield func(string, T) bool) {
		for _, c := range r.items {
			if !yield(c.Name(), c) {
				return
			}
		}
	}
}

func (r *Registry[T]) Names() []string {
	names := make([]string, 0, len(r.items))

	for _, c := range r.items {
		names = append(names, c.Name())
	}

	return names
}

func (r *Registry[T]) Next(name string) T {
	if len(r.items) == 0 {
		var zero T

		return zero
	}

	i, ok := r.index[name]
	if !ok {
		return r.items[0]
	}

	if i == len(r.items)-1 {
		return r.items[0]
	}

	return r.items[i+1]
}
