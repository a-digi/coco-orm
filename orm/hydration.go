package orm

import (
	"database/sql"
	"errors"
	"reflect"
)

type Hydrator struct{}

func (h *Hydrator) HydrateRows(rows *sql.Rows, destSlicePtr interface{}) error {
	destVal := reflect.ValueOf(destSlicePtr)
	if destVal.Kind() != reflect.Ptr || destVal.Elem().Kind() != reflect.Slice {
		return errors.New("destSlicePtr must be a pointer to a slice")
	}

	elemType := destVal.Elem().Type().Elem()
	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	for rows.Next() {
		elem, err := h.Hydrate(rows, elemType, columns)
		if err != nil {
			return err
		}
		destVal.Elem().Set(reflect.Append(destVal.Elem(), elem))
	}

	return rows.Err()
}

func (h *Hydrator) Hydrate(rows *sql.Rows, elemType reflect.Type, columns []string) (reflect.Value, error) {
	elemPtr := reflect.New(elemType)
	elem := elemPtr.Elem()
	fieldPtrs := make([]interface{}, len(columns))
	fieldMap := map[string]int{}

	for i := 0; i < elem.NumField(); i++ {
		tag := elem.Type().Field(i).Tag.Get("db")
		if tag != "" {
			fieldMap[tag] = i
		}
	}

	for i, col := range columns {
		if idx, ok := fieldMap[col]; ok {
			fieldPtrs[i] = elem.Field(idx).Addr().Interface()
		} else {
			var skip interface{}
			fieldPtrs[i] = &skip
		}
	}

	if err := rows.Scan(fieldPtrs...); err != nil {
		return reflect.Value{}, err
	}

	return elem, nil
}
