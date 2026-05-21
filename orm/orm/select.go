package orm

import (
	"fmt"
	"reflect"

	"github.com/a-digi/coco-orm/orm/metadata"
	"github.com/a-digi/coco-orm/orm/querybuilder"
)

func SelectQueryByObject(entity interface{}) (string, error) {
	qb, err := BuildQueryBuilderFromObject(entity)
	if err != nil {
		return "", err
	}

	query := qb.Build()

	return query.SQL, nil
}

func BuildQueryBuilderFromObject(entity interface{}) (*querybuilder.QueryBuilder, error) {
	destType := reflect.TypeOf(entity)

	if destType.Kind() != reflect.Ptr || destType.Elem().Kind() != reflect.Slice {
		return nil, fmt.Errorf("entity must be a pointer to a slice")
	}

	elemType := destType.Elem().Elem()
	tableName, ok := metadata.GetTableName(reflect.New(elemType).Interface())

	if !ok || tableName == "" {
		return nil, fmt.Errorf("table name not found in struct tag")
	}

	dataTypeInfo := metadata.GetDataTypeInfo(reflect.New(elemType).Interface())
	columns := dataTypeInfo.Columns

	if len(columns) == 0 {
		return nil, fmt.Errorf("no db-tagged fields found in struct")
	}

	qb := querybuilder.NewQueryBuilder().Select(columns...).From(tableName)

	if dataTypeInfo.DiscriminatoryCol != "" && dataTypeInfo.DiscriminatoryVal != "" {
		qb = qb.Where(fmt.Sprintf("%s = ?", dataTypeInfo.DiscriminatoryCol), dataTypeInfo.DiscriminatoryVal)
	}

	return qb, nil
}
