package querybuilder

import (
	"fmt"
	"strings"
	"sync"
)

var typeCache sync.Map // map[reflect.Type]struct{columns []string, fieldIndexes []int}

type InsertQueryBuilder struct {
	table   string
	columns []string
	values  []interface{}
}

func (b *InsertQueryBuilder) Into(table string) *InsertQueryBuilder {
	b.table = table
	return b
}

// Columns sets the columns for the INSERT.
func (b *InsertQueryBuilder) Columns(cols ...string) *InsertQueryBuilder {
	if len(b.values) > 0 {
		fmt.Printf("[InsertQueryBuilder] WARNING: Columns called after Values. This may cause mismatches.\n")
	}
	b.columns = append(b.columns, cols...)
	return b
}

func (b *InsertQueryBuilder) Values(vals ...interface{}) *InsertQueryBuilder {
	b.values = vals
	return b
}

func (b *InsertQueryBuilder) Build() (string, []interface{}, error) {

	if b.table == "" {
		return "", nil, fmt.Errorf("table name is required")
	}

	if len(b.columns) == 0 {
		return "", nil, fmt.Errorf("at least one column is required")
	}

	if len(b.values) != len(b.columns) {
		return "", nil, fmt.Errorf("number of values must match number of columns")
	}

	colStr := strings.Join(b.columns, ", ")
	placeholders := make([]string, len(b.values))

	for i := range b.values {
		placeholders[i] = "?"
	}

	phStr := strings.Join(placeholders, ", ")

	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", b.table, colStr, phStr)

	return sql, b.values, nil
}
