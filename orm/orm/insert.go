package orm

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"github.com/google/uuid"

	query_builder "github.com/a-digi/coco-orm/orm/querybuilder"
    "github.com/a-digi/coco-orm/orm/metadata"
)

type InsertObjectQueryBuilder struct {
	cache sync.Map
}

func (b *InsertObjectQueryBuilder) getInsertValue(field reflect.StructField, val reflect.Value, dbType string, defaultValue string) (interface{}, bool) {
	if strings.EqualFold(dbType, "UUID") {
		if val.Kind() == reflect.String && val.IsZero() {
			newUUID := uuid.New().String()
			val.SetString(newUUID)
			return newUUID, true
		}
		return val.Interface(), true
	}

	if strings.EqualFold(dbType, "BOOLEAN") {
		if val.Kind() == reflect.Bool && val.IsZero() && defaultValue != "" {
			if defaultValue == "true" {
				return true, true
			} else if defaultValue == "false" {
				return false, true
			}
		}
		if val.Kind() == reflect.Bool && !val.IsZero() {
			return val.Interface(), true
		}
		if val.IsZero() {
			if defaultValue != "" {
				if defaultValue == "true" {
					return true, true
				} else if defaultValue == "false" {
					return false, true
				}

				return defaultValue, true
			}
			return nil, false
		}
		return val.Interface(), true
	}

	if val.IsZero() {
		if defaultValue != "" {
			switch val.Kind() {
			case reflect.String:
				return defaultValue, true
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				if i, err := parseInt(defaultValue, val.Kind()); err == nil {
					return i, true
				}
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				if u, err := parseUint(defaultValue, val.Kind()); err == nil {
					return u, true
				}
			case reflect.Float32, reflect.Float64:
				if f, err := parseFloat(defaultValue, val.Kind()); err == nil {
					return f, true
				}
			}
			return defaultValue, true // fallback: gib als string zurück
		}
		return nil, false
	}
	return val.Interface(), true
}

// Hilfsfunktionen für Typkonvertierung
func parseInt(s string, kind reflect.Kind) (interface{}, error) {
	var bitSize int
	switch kind {
	case reflect.Int8:
		bitSize = 8
	case reflect.Int16:
		bitSize = 16
	case reflect.Int32:
		bitSize = 32
	case reflect.Int64:
		bitSize = 64
	default:
		bitSize = 0
	}
	v, err := strconv.ParseInt(s, 10, bitSize)
	if err != nil {
		return nil, err
	}
	switch kind {
	case reflect.Int8:
		return int8(v), nil
	case reflect.Int16:
		return int16(v), nil
	case reflect.Int32:
		return int32(v), nil
	case reflect.Int64:
		return int64(v), nil
	default:
		return int(v), nil
	}
}

func parseUint(s string, kind reflect.Kind) (interface{}, error) {
	var bitSize int
	switch kind {
	case reflect.Uint8:
		bitSize = 8
	case reflect.Uint16:
		bitSize = 16
	case reflect.Uint32:
		bitSize = 32
	case reflect.Uint64:
		bitSize = 64
	default:
		bitSize = 0
	}
	v, err := strconv.ParseUint(s, 10, bitSize)
	if err != nil {
		return nil, err
	}
	switch kind {
	case reflect.Uint8:
		return uint8(v), nil
	case reflect.Uint16:
		return uint16(v), nil
	case reflect.Uint32:
		return uint32(v), nil
	case reflect.Uint64:
		return uint64(v), nil
	default:
		return uint(v), nil
	}
}

func parseFloat(s string, kind reflect.Kind) (interface{}, error) {
	bitSize := 64
	if kind == reflect.Float32 {
		bitSize = 32
	}
	v, err := strconv.ParseFloat(s, bitSize)
	if err != nil {
		return nil, err
	}
	if kind == reflect.Float32 {
		return float32(v), nil
	}
	return v, nil
}

func (b *InsertObjectQueryBuilder) setInsertValues(t reflect.Type, v reflect.Value, entry metadata.CacheEntry, dataTypeInfo metadata.DataTypeInfo) ([]string, []interface{}) {

	columns := []string{}
	values := []interface{}{}

	for i, idx := range entry.FieldIndexes {
		field := t.Field(idx)
		val := v.Field(idx)
		var dbType, defaultValue string
		if i < len(dataTypeInfo.Types) {
			dbType = dataTypeInfo.Types[i].Type
			defaultValue = dataTypeInfo.Types[i].Default
		}
		insertVal, ok := b.getInsertValue(field, val, dbType, defaultValue)
		if !ok {
			continue
		}
		values = append(values, insertVal)
		columns = append(columns, field.Tag.Get("db"))
	}

	return columns, values
}

func (b *InsertObjectQueryBuilder) ExtractInsertMetaAndValues(obj interface{}) (string, []string, []interface{}, error) {

typeOfObject, valueOfObject, tableName, cacheEntry, err := metadata.ExtractTypeMeta(
	obj,
	func(t reflect.Type) (metadata.CacheEntry, bool) {
		entry, ok := b.cache.Load(t)
		if !ok {
			return metadata.CacheEntry{}, false
		}
		return entry.(metadata.CacheEntry), true
	},
	func(t reflect.Type, entry metadata.CacheEntry) {
		b.cache.Store(t, entry)
	},
)
if err != nil {
	return "", nil, nil, err
}

dataTypeInfo := metadata.GetDataTypeInfo(obj)
columns, values := b.setInsertValues(typeOfObject, valueOfObject, cacheEntry, dataTypeInfo)

if len(columns) == 0 {
	return "", nil, nil, fmt.Errorf("no columns with values found for insert")
}

if len(columns) != len(values) {
	return "", nil, nil, fmt.Errorf("number of columns (%d) does not match number of values (%d)", len(columns), len(values))
}

return tableName, columns, values, nil
}

func (b *InsertObjectQueryBuilder) BuildFrom(obj interface{}) (string, []interface{}, error) {
	tableName, columns, values, err := b.ExtractInsertMetaAndValues(obj)
	if err != nil {
		return "", nil, err
	}

	builder := &query_builder.InsertQueryBuilder{}
	builder.Into(tableName).Columns(columns...).Values(values...)

	return builder.Build()
}
