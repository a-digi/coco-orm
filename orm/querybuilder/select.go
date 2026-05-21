package querybuilder

import (
	"fmt"
	"strings"
	"time"
)

const (
	DefaultUnsetLimit  = -1
	DefaultUnsetOffset = -1
)

type QueryBuilder struct {
	selectCols   []string
	table        string
	whereClauses []string
	orClauses    []string
	orderClauses []string
	limit        int
	offset       int
	args         []interface{}

	distinct       bool
	joins          []string
	groupByClauses []string
	havingClauses  []string
}

type Query struct {
	SQL  string
	Args []interface{}
}

func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{
		limit:  DefaultUnsetLimit,
		offset: DefaultUnsetOffset,
	}
}

func (qb *QueryBuilder) Select(cols ...string) *QueryBuilder {
	qb.selectCols = append(qb.selectCols, cols...)

	return qb
}

func (qb *QueryBuilder) From(table string) *QueryBuilder {
	qb.table = table

	return qb
}

func (qb *QueryBuilder) Where(condition string, args ...interface{}) *QueryBuilder {
	qb.whereClauses = append(qb.whereClauses, condition)
	qb.args = append(qb.args, args...)

	return qb
}

func (qb *QueryBuilder) OrWhere(condition string, args ...interface{}) *QueryBuilder {
	qb.orClauses = append(qb.orClauses, condition)
	qb.args = append(qb.args, args...)
	return qb
}

func (qb *QueryBuilder) WhereGte(col string, val interface{}) *QueryBuilder {
	qb.whereClauses = append(qb.whereClauses, fmt.Sprintf("%s >= ?", col))
	qb.args = append(qb.args, val)
	return qb
}

func (qb *QueryBuilder) WhereLte(col string, val interface{}) *QueryBuilder {
	qb.whereClauses = append(qb.whereClauses, fmt.Sprintf("%s <= ?", col))
	qb.args = append(qb.args, val)
	return qb
}

func (qb *QueryBuilder) WhereDateGte(col string, val time.Time) *QueryBuilder {
	qb.whereClauses = append(qb.whereClauses, fmt.Sprintf("datetime(%s) >= datetime(?)", col))
	qb.args = append(qb.args, val.UTC().Format("2006-01-02T15:04:05Z"))
	return qb
}

func (qb *QueryBuilder) WhereDateLte(col string, val time.Time) *QueryBuilder {
	qb.whereClauses = append(qb.whereClauses, fmt.Sprintf("datetime(%s) <= datetime(?)", col))
	qb.args = append(qb.args, val.UTC().Format("2006-01-02T15:04:05Z"))
	return qb
}

func (qb *QueryBuilder) WhereLike(col string, val string) *QueryBuilder {
	qb.whereClauses = append(qb.whereClauses, fmt.Sprintf("%s LIKE ?", col))
	qb.args = append(qb.args, val)
	return qb
}

func (qb *QueryBuilder) OrderBy(clauses ...string) *QueryBuilder {
	qb.orderClauses = append(qb.orderClauses, clauses...)

	return qb
}

func (qb *QueryBuilder) Limit(limit int) *QueryBuilder {
	qb.limit = limit
	return qb
}

func (qb *QueryBuilder) Offset(offset int) *QueryBuilder {
	qb.offset = offset

	return qb
}

func (qb *QueryBuilder) Distinct() *QueryBuilder {
	qb.distinct = true

	return qb
}

func (qb *QueryBuilder) Join(joinClause string) *QueryBuilder {
	qb.joins = append(qb.joins, "JOIN "+joinClause)
	return qb
}

func (qb *QueryBuilder) LeftJoin(joinClause string) *QueryBuilder {
	qb.joins = append(qb.joins, "LEFT JOIN "+joinClause)
	return qb
}

func (qb *QueryBuilder) RightJoin(joinClause string) *QueryBuilder {
	qb.joins = append(qb.joins, "RIGHT JOIN "+joinClause)
	return qb
}

func (qb *QueryBuilder) GroupBy(cols ...string) *QueryBuilder {
	qb.groupByClauses = append(qb.groupByClauses, cols...)
	return qb
}

func (qb *QueryBuilder) Having(condition string, args ...interface{}) *QueryBuilder {
	qb.havingClauses = append(qb.havingClauses, condition)
	qb.args = append(qb.args, args...)
	return qb
}

// Build constructs the Query struct
func (qb *QueryBuilder) Build() Query {
	var sb strings.Builder
	if qb.distinct {
		sb.WriteString("SELECT DISTINCT ")
	} else if len(qb.selectCols) == 0 {
		sb.WriteString("SELECT *")
	} else {
		sb.WriteString("SELECT ")
	}
	if !qb.distinct && len(qb.selectCols) > 0 {
		sb.WriteString(strings.Join(qb.selectCols, ", "))
	}

	if qb.table != "" {
		sb.WriteString(" FROM ")
		sb.WriteString(qb.table)
	}

	if len(qb.joins) > 0 {
		sb.WriteString(" ")
		sb.WriteString(strings.Join(qb.joins, " "))
	}

	// WHERE and OR logic
	where := ""
	if len(qb.whereClauses) > 0 && len(qb.orClauses) > 0 {
		where = "(" + strings.Join(qb.whereClauses, " AND ") + ") OR (" + strings.Join(qb.orClauses, " OR ") + ")"
	} else if len(qb.whereClauses) > 0 {
		where = strings.Join(qb.whereClauses, " AND ")
	} else if len(qb.orClauses) > 0 {
		where = strings.Join(qb.orClauses, " OR ")
	}
	if where != "" {
		sb.WriteString(" WHERE ")
		sb.WriteString(where)
	}

	if len(qb.groupByClauses) > 0 {
		sb.WriteString(" GROUP BY ")
		sb.WriteString(strings.Join(qb.groupByClauses, ", "))
	}

	if len(qb.havingClauses) > 0 {
		sb.WriteString(" HAVING ")
		sb.WriteString(strings.Join(qb.havingClauses, " AND "))
	}

	if len(qb.orderClauses) > 0 {
		sb.WriteString(" ORDER BY ")
		sb.WriteString(strings.Join(qb.orderClauses, ", "))
	}

	if qb.limit != DefaultUnsetLimit {
		sb.WriteString(fmt.Sprintf(" LIMIT %d", qb.limit))
	}

	if qb.offset != DefaultUnsetOffset {
		sb.WriteString(fmt.Sprintf(" OFFSET %d", qb.offset))
	}

	return Query{
		SQL:  sb.String(),
		Args: qb.args,
	}
}

// ToSQL returns the SQL string and arguments for the query
func (q Query) ToSQL() (string, []interface{}) {
	return q.SQL, q.Args
}
