package metadata

import (
	"reflect"
	"sync"
)

var tableNameCache sync.Map

func GetTableName(obj interface{}) (string, bool) {
	t := reflect.TypeOf(obj)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if cached, ok := tableNameCache.Load(t); ok {
		return cached.(string), true
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if tableTag, ok := field.Tag.Lookup("table"); ok {
			tableNameCache.Store(t, tableTag)
			return tableTag, true
		}
	}

	return "", false
}
