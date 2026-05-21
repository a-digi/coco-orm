package orm

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/a-digi/coco-orm/orm/metadata"
	query_builder "github.com/a-digi/coco-orm/orm/querybuilder"
)

// IdentityBag represents a blueprint to identify which database entries should be updated.
// It maps column names to their expected values.
type IdentityBag map[string]interface{}

type UpdateObjectQueryBuilder struct {
	cache sync.Map
}

func (b *UpdateObjectQueryBuilder) getUpdateValue(val reflect.Value, dbType string, defaultValue string) (interface{}, bool) {
	// For UPDATE, we usually don't want to generate new UUIDs if they are empty.
	if strings.EqualFold(dbType, "UUID") && val.Kind() == reflect.String {
		if val.IsZero() {
			return nil, false // Skip updating empty UUIDs
		}
		return val.Interface(), true
	}

	if strings.EqualFold(dbType, "BOOLEAN") || val.Kind() == reflect.Bool {
		return val.Interface(), true
	}

	if val.IsZero() {
		return nil, false
	}
	return val.Interface(), true
}

func (b *UpdateObjectQueryBuilder) setUpdateValues(t reflect.Type, v reflect.Value, entry metadata.CacheEntry, dataTypeInfo metadata.DataTypeInfo) ([]string, []interface{}) {
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
		updateVal, ok := b.getUpdateValue(val, dbType, defaultValue)
		if !ok {
			continue
		}
		columns = append(columns, field.Tag.Get("db"))
		values = append(values, updateVal)
	}

	return columns, values
}

func (b *UpdateObjectQueryBuilder) ExtractUpdateMetaAndValues(obj interface{}) (string, []string, []interface{}, error) {
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
	columns, values := b.setUpdateValues(typeOfObject, valueOfObject, cacheEntry, dataTypeInfo)

	if len(columns) == 0 {
		return "", nil, nil, fmt.Errorf("no columns with values found for update")
	}

	if len(columns) != len(values) {
		return "", nil, nil, fmt.Errorf("number of columns (%d) does not match number of values (%d)", len(columns), len(values))
	}

	return tableName, columns, values, nil
}

// BuildFrom generates an UPDATE query.
// It requires an IdentityBag (blueprint) to identify the specific entries to update and prevent data corruption.
func (b *UpdateObjectQueryBuilder) BuildFrom(obj interface{}, identity IdentityBag) (string, []interface{}, error) {
	if len(identity) == 0 {
		return "", nil, fmt.Errorf("an identity bag must be provided to block unregulated updates to the database")
	}

	tableName, columns, values, err := b.ExtractUpdateMetaAndValues(obj)
	if err != nil {
		return "", nil, err
	}

	builder := &query_builder.UpdateQueryBuilder{}
	builder.Table(tableName).Columns(columns...).Values(values...)

	for col, val := range identity {
		builder.Where(fmt.Sprintf("%s = ?", col), val)
	}

	return builder.Build()
}
