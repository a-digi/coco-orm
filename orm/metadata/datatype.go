package metadata

import (
	"reflect"
	"sync"
)

type DataType struct {
	Type           string
	Nullable       bool
	Default        string
	Discriminatory string
}

type RelationInfo struct {
	Type       string // e.g. "m2m"
	FieldType  reflect.Type
	FieldName  string
	JoinEntity string
	JoinTable  string
	JoinFK     string
	JoinAssFK  string
}

type DataTypeInfo struct {
	Columns           []string
	Types             []DataType
	DiscriminatoryCol string
	DiscriminatoryVal string
	Relations         []RelationInfo
}

var dataTypeCache sync.Map

func GetDataTypeInfo(obj interface{}) DataTypeInfo {
	t := reflect.TypeOf(obj)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return DataTypeInfo{}
	}

	if cached, ok := dataTypeCache.Load(t); ok {
		return cached.(DataTypeInfo)
	}

	columns := []string{}
	types := []DataType{}
	relations := []RelationInfo{}
	discriminatoryCol := ""
	discriminatoryVal := ""
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}
		dbTag := field.Tag.Get("db")
		dbType := field.Tag.Get("dbtype")
		nullableTag := field.Tag.Get("nullable")
		defaultTag := ""
		if tag := field.Tag.Get("default"); tag != "" {
			defaultTag = tag
		}
		// discriminatory-Tag extrahieren
		discriminatoryTag := field.Tag.Get("discriminatory")

		relationTag := field.Tag.Get("relation")
		if relationTag != "" {
			relations = append(relations, RelationInfo{
				Type:       relationTag,
				FieldType:  field.Type,
				FieldName:  field.Name,
				JoinEntity: field.Tag.Get("join_entity"),
				JoinTable:  field.Tag.Get("join_table"),
				JoinFK:     field.Tag.Get("join_fk"),
				JoinAssFK:  field.Tag.Get("join_ass_fk"),
			})
			continue
		}

		if dbTag == "" || dbTag == "-" {
			continue
		}
		nullable := false
		if nullableTag == "true" {
			nullable = true
		}
		columns = append(columns, dbTag)
		types = append(types, DataType{Type: dbType, Nullable: nullable, Default: defaultTag, Discriminatory: discriminatoryTag})
		if discriminatoryTag != "" && discriminatoryCol == "" {
			discriminatoryCol = dbTag
			discriminatoryVal = discriminatoryTag
		}
	}

	info := DataTypeInfo{Columns: columns, Types: types, DiscriminatoryCol: discriminatoryCol, DiscriminatoryVal: discriminatoryVal, Relations: relations}
	dataTypeCache.Store(t, info)

	return info
}
