package registry

import (
	"iter"
)

type (
	Option[T Registrable] func(*options[T])
)

type options[T Registrable] struct {
	producers []iter.Seq[T]
}

func With[T Registrable](producers ...iter.Seq[T]) Option[T] {
	return func(o *options[T]) {
		o.producers = append(o.producers, producers...)
	}
}
