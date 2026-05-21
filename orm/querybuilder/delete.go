package querybuilder

import (
	"fmt"
	"strings"
)

type DeleteQueryBuilder struct {
	table        string
	whereClauses []string
	orClauses    []string
	orderClauses []string
	limit        int
	offset       int
	args         []interface{}
}

func NewDeleteQueryBuilder() *DeleteQueryBuilder {
	return &DeleteQueryBuilder{
		limit:  DefaultUnsetLimit,
		offset: DefaultUnsetOffset,
	}
}

func (qb *DeleteQueryBuilder) From(table string) *DeleteQueryBuilder {
	qb.table = table
	return qb
}

func (qb *DeleteQueryBuilder) Where(condition string, args ...interface{}) *DeleteQueryBuilder {
	qb.whereClauses = append(qb.whereClauses, condition)
	qb.args = append(qb.args, args...)
	return qb
}

func (qb *DeleteQueryBuilder) OrWhere(condition string, args ...interface{}) *DeleteQueryBuilder {
	qb.orClauses = append(qb.orClauses, condition)
	qb.args = append(qb.args, args...)
	return qb
}

func (qb *DeleteQueryBuilder) OrderBy(clauses ...string) *DeleteQueryBuilder {
	qb.orderClauses = append(qb.orderClauses, clauses...)
	return qb
}

func (qb *DeleteQueryBuilder) Limit(limit int) *DeleteQueryBuilder {
	qb.limit = limit
	return qb
}

func (qb *DeleteQueryBuilder) Offset(offset int) *DeleteQueryBuilder {
	qb.offset = offset
	return qb
}

// Build constructs the Query struct
func (qb *DeleteQueryBuilder) Build() (Query, error) {
	if qb.table == "" {
		return Query{}, fmt.Errorf("table name is required for delete query")
	}

	var sb strings.Builder

	sb.WriteString("DELETE FROM ")
	sb.WriteString(qb.table)

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
	}, nil
}
