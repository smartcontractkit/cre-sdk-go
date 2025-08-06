package cre

import (
	"sync"
)

// Promise allows for asynchronous computation, where the result can be awaited later.
type Promise[T any] interface {
	// Await blocks until the computation is complete and returns the result or an error.
	Await() (T, error)
}

// NewBasicPromise is meant to be used by Runtime implementations.
// It creates a new Promise that executes the provided function once awaited.
// This provides asynchronous callbacks in single-threaded guest environments as the host can execute while the guest continues to run.
func NewBasicPromise[T any](await func() (T, error)) Promise[T] {
	return &basicPromise[T]{await: sync.OnceValues(await)}
}

// PromiseFromResult creates a Promise that immediately returns the provided result and error.
func PromiseFromResult[T any](result T, err error) Promise[T] {
	return &basicPromise[T]{await: func() (T, error) { return result, err }}
}

type basicPromise[T any] struct {
	await func() (T, error)
}

func (t *basicPromise[T]) Await() (T, error) {
	return t.await()
}

func (t *basicPromise[T]) promise() {}

// Then allows chaining of promises, upon success, the result of the first promise is passed to the provided function.
// Note that the callback in `fn` will not be executed until Await is called on the returned Promise.
func Then[I, O any](p Promise[I], fn func(I) (O, error)) Promise[O] {
	return NewBasicPromise[O](func() (O, error) {
		underlyingResult, err := p.Await()
		if err != nil {
			var o O
			return o, err
		}

		return fn(underlyingResult)
	})
}

// ThenPromise allows chaining of promises, similar to Then, but the provided function returns a Promise.
// This is useful when the next step in the chain is also asynchronous.
func ThenPromise[I, O any](p Promise[I], fn func(I) Promise[O]) Promise[O] {
	return NewBasicPromise[O](func() (O, error) {
		underlyingResult, err := p.Await()
		if err != nil {
			var o O
			return o, err
		}
		return fn(underlyingResult).Await()
	})
}
