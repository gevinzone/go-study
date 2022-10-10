
package orm

import (
	"context"
)

type Updater[T any] struct {
	builder
	db      *DB
	assigns []Assignable
	val     *T
	where   []Predicate
}

func NewUpdater[T any](db *DB) *Updater[T] {
	panic("implement me")
}

func (u *Updater[T]) Update(t *T) *Updater[T] {
	panic("implement me")
}

func (u *Updater[T]) Set(assigns ...Assignable) *Updater[T] {
	panic("implement me")
}

func (u *Updater[T]) Build() (*Query, error) {
	panic("implement me")
}


func (u *Updater[T]) Where(ps ...Predicate) *Updater[T] {
	panic("implement me")
}

func (u *Updater[T]) Exec(ctx context.Context) Result {
	panic("implement me")
}

// AssignNotZeroColumns 更新非零值
func AssignNotZeroColumns(entity interface{}) []Assignable {
	panic("implement me")
}