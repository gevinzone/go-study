package orm

import (
	"context"
	"gitee.com/geektime-geekbang/geektime-go/orm/homework2/internal/errs"
	"reflect"
)

type Updater[T any] struct {
	builder
	db      *DB
	assigns []Assignable
	val     *T
	where   []Predicate
}

func NewUpdater[T any](db *DB) *Updater[T] {
	return &Updater[T]{
		builder: builder{
			dialect: db.dialect,
			quoter:  db.dialect.quoter(),
		},
		db: db,
	}
}

func (u *Updater[T]) Update(t *T) *Updater[T] {
	u.val = t
	return u
}

func (u *Updater[T]) Set(assigns ...Assignable) *Updater[T] {
	u.assigns = assigns
	return u
}

func (u *Updater[T]) Build() (*Query, error) {
	if len(u.assigns) == 0 {
		return nil, errs.ErrNoUpdatedColumns
	}
	var (
		err error
		t   T
	)
	u.model, err = u.db.r.Get(&t)
	if err != nil {
		return nil, err
	}

	u.sb.WriteString("UPDATE ")
	u.quote(u.model.TableName)
	u.sb.WriteString(" SET ")

	if err = u.buildAssignables(); err != nil {
		return nil, err
	}

	if len(u.where) > 0 {
		u.sb.WriteString(" WHERE ")
		if err = u.buildPredicates(u.where); err != nil {
			return nil, err
		}
	}
	u.sb.WriteByte(';')

	return &Query{
		SQL:  u.sb.String(),
		Args: u.args,
	}, nil
}

func (u *Updater[T]) buildAssignables() error {
	val := u.db.valCreator(u.val, u.model)
	for i, a := range u.assigns {
		if i > 0 {
			u.sb.WriteByte(',')
		}
		switch assign := a.(type) {
		case Column:
			if err := u.buildColumn(assign.name); err != nil {
				return err
			}
			u.sb.WriteString("=?")
			arg, err := val.Field(assign.name)
			if err != nil {
				return err
			}
			u.addArgs(arg)
		case Assignment:
			if err := u.buildAssignment(assign); err != nil {
				return err
			}
		default:
			return errs.NewErrUnsupportedAssignableType(a)
		}
	}
	return nil
}

func (u *Updater[T]) buildAssignment(assign Assignment) error {
	if err := u.buildColumn(assign.column); err != nil {
		return err
	}
	u.sb.WriteByte('=')
	return u.buildExpression(assign.val)
}

func (u *Updater[T]) Where(ps ...Predicate) *Updater[T] {
	u.where = ps
	return u
}

func (u *Updater[T]) Exec(ctx context.Context) Result {
	q, err := u.Build()
	if err != nil {
		return Result{err: err}
	}
	res, err := u.db.db.ExecContext(ctx, q.SQL, q.Args...)
	return Result{err: err, res: res}
}

// AssignNotZeroColumns 更新非零值
func AssignNotZeroColumns(entity interface{}) []Assignable {
	typ := reflect.TypeOf(entity)
	val := reflect.ValueOf(entity)

	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
		val = val.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return []Assignable{}
	}
	n := typ.NumField()
	res := make([]Assignable, 0, n)
	for i := 0; i < n; i++ {
		t := typ.Field(i)
		f := val.Field(i)
		if !f.IsZero() {
			assign := Assignment{
				column: t.Name,
				val:    valueOf(f.Interface()),
			}
			res = append(res, assign)
		}
	}

	return res
}
