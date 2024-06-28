package cache

import "container/ring"

type Ring[T any] ring.Ring

func New[T any](size int) *Ring[T] {
	return (*Ring[T])(ring.New(size))
}

func (r *Ring[T]) Next() *Ring[T] {
	return (*Ring[T])((*ring.Ring)(r).Next())
}

func (r *Ring[T]) Prev() *Ring[T] {
	return (*Ring[T])((*ring.Ring)(r).Prev())

}
func (r *Ring[T]) Move(n int) *Ring[T] {
	return (*Ring[T])((*ring.Ring)(r).Move(n))
}

func (r *Ring[T]) Link(s *Ring[T]) *Ring[T] {
	return (*Ring[T])((*ring.Ring)(r).Link((*ring.Ring)(s)))
}

func (r *Ring[T]) Unlink(n int) *Ring[T] {
	return (*Ring[T])((*ring.Ring)(r).Unlink(n))
}

func (r *Ring[T]) Len() int {
	return (*ring.Ring)(r).Len()
}

func (r *Ring[T]) Do(f func(T)) {
	(*ring.Ring)(r).Do(func(a any) {
		f(a.(T))
	})
}
