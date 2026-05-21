package metadata

import (
	"fmt"
	"reflect"
)

// CacheEntry wird für die Metadaten verwendet
// (ggf. in ein gemeinsames Modell auslagern, falls von mehreren Paketen benötigt)
type CacheEntry struct {
	Columns      []string
	FieldIndexes []int
}

// ExtractTypeMeta extrahiert Typ-, Value-, Tabellen- und Cache-Metadaten für ein Objekt
func ExtractTypeMeta(obj interface{}, cacheLoader func(reflect.Type) (CacheEntry, bool), cacheStorer func(reflect.Type, CacheEntry)) (reflect.Type, reflect.Value, string, CacheEntry, error) {
	typeOfObject := reflect.TypeOf(obj)
	valueOfObject := reflect.ValueOf(obj)
	if typeOfObject.Kind() == reflect.Ptr {
		typeOfObject = typeOfObject.Elem()
		valueOfObject = valueOfObject.Elem()
	}
	if typeOfObject.Kind() != reflect.Struct {
		return nil, reflect.Value{}, "", CacheEntry{}, fmt.Errorf("object must be a struct or pointer to struct")
	}

	tableName, hasTableName := GetTableName(obj)
	if !hasTableName || tableName == "" {
		return nil, reflect.Value{}, "", CacheEntry{}, fmt.Errorf("table name not found in struct tag")
	}

	var cacheEntry CacheEntry
	if cached, ok := cacheLoader(typeOfObject); ok {
		cacheEntry = cached
	} else {
		columns := []string{}
		fieldIndexes := []int{}
		for i := 0; i < typeOfObject.NumField(); i++ {
			field := typeOfObject.Field(i)
			if !field.IsExported() {
				continue
			}
			dbTag := field.Tag.Get("db")
			if dbTag == "" || dbTag == "-" {
				continue
			}
			columns = append(columns, dbTag)
			fieldIndexes = append(fieldIndexes, i)
		}
		cacheEntry = CacheEntry{Columns: columns, FieldIndexes: fieldIndexes}
		cacheStorer(typeOfObject, cacheEntry)
	}

	if len(cacheEntry.Columns) == 0 {
		return nil, reflect.Value{}, "", CacheEntry{}, fmt.Errorf("no db-tagged fields found in struct")
	}

	return typeOfObject, valueOfObject, tableName, cacheEntry, nil
}
