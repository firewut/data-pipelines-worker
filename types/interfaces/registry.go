package interfaces

import "context"

// Registry is a generic interface that works with any type T.
type Registry[T any] interface {
	Add(T)

	Get(string) T
	GetAll() map[string]T

	Delete(string)
	DeleteAll()

	Shutdown(context.Context) error
}
