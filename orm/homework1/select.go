package orm

import (
	"context"
	"database/sql"
	"gitee.com/geektime-geekbang/geektime-go/orm/homework1/internal/errs"
	"gitee.com/geektime-geekbang/geektime-go/orm/homework1/model"
	"strings"
)

// Selector 用于构造 SELECT 语句
type Selector[T any] struct {
	sb      strings.Builder
	args    []any
	table   string
	where   []Predicate
	having  []Predicate
	model   *model.Model
	db      *DB
	columns []Selectable
	groupBy []Column
	orderBy []OrderBy
	offset  int
	limit   int
}

func (s *Selector[T]) Select(cols ...Selectable) *Selector[T] {
	s.columns = cols
	return s
}

// From 指定表名，如果是空字符串，那么将会使用默认表名
func (s *Selector[T]) From(tbl string) *Selector[T] {
	s.table = tbl
	return s
}

func (s *Selector[T]) Build() (*Query, error) {
	var (
		t   T
		err error
	)
	s.model, err = s.db.r.Get(&t)
	if err != nil {
		return nil, err
	}
	s.sb.WriteString("SELECT ")
	if err = s.buildColumns(); err != nil {
		return nil, err
	}
	s.sb.WriteString(" FROM ")
	if s.table == "" {
		s.sb.WriteByte('`')
		s.sb.WriteString(s.model.TableName)
		s.sb.WriteByte('`')
	} else {
		s.sb.WriteString(s.table)
	}

	if err = s.buildWhere(); err != nil {
		return nil, err
	}
	if err = s.buildGroupBy(); err != nil {
		return nil, err
	}
	if err = s.buildHaving(); err != nil {
		return nil, err
	}
	if err = s.buildOrderBy(); err != nil {
		return nil, err
	}
	if err = s.buildLimit(); err != nil {
		return nil, err
	}
	if err = s.buildOffset(); err != nil {
		return nil, err
	}

	s.sb.WriteString(";")
	return &Query{
		SQL:  s.sb.String(),
		Args: s.args,
	}, nil
}

func (s *Selector[T]) buildWhere() error {
	if len(s.where) < 1 {
		return nil
	}
	// 类似这种可有可无的部分，都要在前面加一个空格
	s.sb.WriteString(" WHERE ")
	// WHERE 是不允许用别名的
	return s.buildPredicates(s.where)
}

func (s *Selector[T]) buildGroupBy() error {
	if len(s.groupBy) < 1 {
		return nil
	}
	s.sb.WriteString(" GROUP BY ")
	for i, c := range s.groupBy {
		if i > 0 {
			s.sb.WriteByte(',')
		}
		if err := s.buildColumn(c.name, c.alias); err != nil {
			return err
		}
	}
	return nil
}

func (s *Selector[T]) buildHaving() error {
	if len(s.having) < 1 {
		return nil
	}
	s.sb.WriteString(" HAVING ")
	return s.buildPredicates(s.having)
}

func (s *Selector[T]) buildOrderBy() error {
	if len(s.orderBy) < 1 {
		return nil
	}
	s.sb.WriteString(" ORDER BY ")
	for idx, ob := range s.orderBy {
		if idx > 0 {
			s.sb.WriteByte(',')
		}
		err := s.buildColumn(ob.col, "")
		if err != nil {
			return err
		}
		s.sb.WriteByte(' ')
		s.sb.WriteString(ob.order)
	}
	return nil
}

func (s *Selector[T]) buildLimit() error {
	if s.limit == 0 {
		return nil
	}
	s.sb.WriteString(" LIMIT ")
	s.sb.WriteByte('?')
	s.args = append(s.args, s.limit)

	return nil
}

func (s *Selector[T]) buildOffset() error {
	if s.offset == 0 {
		return nil
	}
	s.sb.WriteString(" OFFSET ")
	s.sb.WriteByte('?')
	s.args = append(s.args, s.offset)

	return nil
}

func (s *Selector[T]) buildPredicates(ps []Predicate) error {
	p := ps[0]
	for i := 1; i < len(ps); i++ {
		p = p.And(ps[i])
	}
	return s.buildExpression(p)
}

func (s *Selector[T]) buildColumns() error {
	if len(s.columns) == 0 {
		s.sb.WriteByte('*')
		return nil
	}
	for i, c := range s.columns {
		if i > 0 {
			s.sb.WriteByte(',')
		}
		switch val := c.(type) {
		case Column:
			if err := s.buildColumn(val.name, val.alias); err != nil {
				return err
			}
		case Aggregate:
			if err := s.buildAggregate(val, true); err != nil {
				return err
			}
		case RawExpr:
			s.sb.WriteString(val.raw)
			if len(val.args) != 0 {
				s.addArgs(val.args...)
			}
		default:
			return errs.NewErrUnsupportedSelectable(c)
		}
	}
	return nil
}

func (s *Selector[T]) buildAggregate(a Aggregate, useAlias bool) error {
	s.sb.WriteString(a.fn)
	s.sb.WriteString("(`")
	fd, ok := s.model.FieldMap[a.arg]
	if !ok {
		return errs.NewErrUnknownField(a.arg)
	}
	s.sb.WriteString(fd.ColName)
	s.sb.WriteString("`)")
	if useAlias {
		s.buildAs(a.alias)
	}
	return nil
}

func (s *Selector[T]) buildColumn(c string, alias string) error {
	s.sb.WriteByte('`')
	fd, ok := s.model.FieldMap[c]
	if !ok {
		return errs.NewErrUnknownField(c)
	}
	s.sb.WriteString(fd.ColName)
	s.sb.WriteByte('`')
	if alias != "" {
		s.buildAs(alias)
	}
	return nil
}

func (s *Selector[T]) buildExpression(e Expression) error {
	if e == nil {
		return nil
	}
	switch exp := e.(type) {
	case Column:
		if err := s.buildColumn(exp.name, ""); err != nil {
			return err
		}
	case value:
		s.sb.WriteByte('?')
		s.args = append(s.args, exp.val)
	case Predicate:
		_, lp := exp.left.(Predicate)
		if lp {
			s.sb.WriteByte('(')
		}
		if err := s.buildExpression(exp.left); err != nil {
			return err
		}
		if lp {
			s.sb.WriteByte(')')
		}

		if exp.op.String() != "" {
			s.sb.WriteByte(' ')
			s.sb.WriteString(exp.op.String())
			s.sb.WriteByte(' ')
		}

		_, rp := exp.right.(Predicate)
		if rp {
			s.sb.WriteByte('(')
		}
		if err := s.buildExpression(exp.right); err != nil {
			return err
		}
		if rp {
			s.sb.WriteByte(')')
		}
	case Aggregate:
		if err := s.buildAggregate(exp, false); err != nil {
			return err
		}
	case RawExpr:
		s.sb.WriteString(exp.raw)
		if len(exp.args) != 0 {
			s.addArgs(exp.args...)
		}

	default:
		return errs.NewErrUnsupportedExpressionType(exp)
	}
	return nil
}

// Where 用于构造 WHERE 查询条件。如果 ps 长度为 0，那么不会构造 WHERE 部分
func (s *Selector[T]) Where(ps ...Predicate) *Selector[T] {
	s.where = ps
	return s
}

// GroupBy 设置 group by 子句
func (s *Selector[T]) GroupBy(cols ...Column) *Selector[T] {
	s.groupBy = cols
	return s
}

func (s *Selector[T]) Having(ps ...Predicate) *Selector[T] {
	s.having = ps
	return s
}

func (s *Selector[T]) Offset(offset int) *Selector[T] {
	s.offset = offset
	return s
}

func (s *Selector[T]) Limit(limit int) *Selector[T] {
	s.limit = limit
	return s
}

func (s *Selector[T]) OrderBy(orderBys ...OrderBy) *Selector[T] {
	s.orderBy = orderBys
	return s
}

func (s *Selector[T]) Get(ctx context.Context) (*T, error) {
	q, err := s.Build()
	if err != nil {
		return nil, err
	}
	// s.db 是我们定义的 DB
	// s.db.db 则是 sql.DB
	// 使用 QueryContext，从而和 GetMulti 能够复用处理结果集的代码
	rows, err := s.db.db.QueryContext(ctx, q.SQL, q.Args...)
	if err != nil {
		return nil, err
	}

	if !rows.Next() {
		return nil, ErrNoRows
	}

	tp := new(T)
	meta, err := s.db.r.Get(tp)
	if err != nil {
		return nil, err
	}
	val := s.db.valCreator(tp, meta)
	err = val.SetColumns(rows)
	return tp, err
}

func (s *Selector[T]) addArgs(args ...any) {
	if s.args == nil {
		s.args = make([]any, 0, 8)
	}
	s.args = append(s.args, args...)
}

func (s *Selector[T]) buildAs(alias string) {
	if alias != "" {
		s.sb.WriteString(" AS ")
		s.sb.WriteByte('`')
		s.sb.WriteString(alias)
		s.sb.WriteByte('`')
	}
}

func (s *Selector[T]) GetMulti(ctx context.Context) ([]*T, error) {
	var db sql.DB
	q, err := s.Build()
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, q.SQL, q.Args...)
	if err != nil {
		return nil, err
	}

	res := make([]*T, 0, 10)
	for rows.Next() {
		t := new(T)
		meta, err := s.db.r.Get(t)
		if err != nil {
			return nil, err
		}
		val := s.db.valCreator(t, meta)
		err = val.SetColumns(rows)
		if err != nil {
			return nil, err
		}
		res = append(res, t)
	}
	return res, nil
}

func NewSelector[T any](db *DB) *Selector[T] {
	return &Selector[T]{
		db: db,
	}
}

type Selectable interface {
	selectable()
}

type OrderBy struct {
	col   string
	order string
}

func Asc(col string) OrderBy {
	return OrderBy{
		col:   col,
		order: "ASC",
	}
}

func Desc(col string) OrderBy {
	return OrderBy{
		col:   col,
		order: "DESC",
	}
}
