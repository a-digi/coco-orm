package querybuilder

import (
	"fmt"
	"strings"
)

type UpdateQueryBuilder struct {
	table        string
	columns      []string
	values       []interface{}
	whereClauses []string
	whereArgs    []interface{}
}

func (b *UpdateQueryBuilder) Table(table string) *UpdateQueryBuilder {
	b.table = table
	return b
}

// Columns sets the columns for the UPDATE.
func (b *UpdateQueryBuilder) Columns(cols ...string) *UpdateQueryBuilder {
	if len(b.values) > 0 {
		fmt.Printf("[UpdateQueryBuilder] WARNING: Columns called after Values. This may cause mismatches.\n")
	}
	b.columns = append(b.columns, cols...)
	return b
}

// Values sets the values for the UPDATE.
func (b *UpdateQueryBuilder) Values(vals ...interface{}) *UpdateQueryBuilder {
	b.values = append(b.values, vals...)
	return b
}

// Set adds a single column/value pair to the UPDATE.
func (b *UpdateQueryBuilder) Set(col string, val interface{}) *UpdateQueryBuilder {
	b.columns = append(b.columns, col)
	b.values = append(b.values, val)
	return b
}

// Where adds a WHERE condition.
func (b *UpdateQueryBuilder) Where(condition string, args ...interface{}) *UpdateQueryBuilder {
	b.whereClauses = append(b.whereClauses, condition)
	b.whereArgs = append(b.whereArgs, args...)
	return b
}

func (b *UpdateQueryBuilder) Build() (string, []interface{}, error) {
	if b.table == "" {
		return "", nil, fmt.Errorf("table name is required")
	}

	if len(b.columns) == 0 {
		return "", nil, fmt.Errorf("at least one column is required")
	}

	if len(b.values) != len(b.columns) {
		return "", nil, fmt.Errorf("number of values must match number of columns")
	}

	setParts := make([]string, len(b.columns))
	for i, col := range b.columns {
		setParts[i] = fmt.Sprintf("%s = ?", col)
	}
	setStr := strings.Join(setParts, ", ")

	sql := fmt.Sprintf("UPDATE %s SET %s", b.table, setStr)

	var allArgs []interface{}
	// Copy the set values
	allArgs = append(allArgs, b.values...)

	// Append WHERE conditions
	if len(b.whereClauses) > 0 {
		whereStr := strings.Join(b.whereClauses, " AND ")
		sql = fmt.Sprintf("%s WHERE %s", sql, whereStr)
		allArgs = append(allArgs, b.whereArgs...)
	}

	return sql, allArgs, nil
}
