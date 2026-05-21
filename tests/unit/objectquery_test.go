package unit

import (
	"testing"

	"github.com/a-digi/coco-orm/orm/metadata"
	"github.com/a-digi/coco-orm/orm/orm"
)

type testStruct struct {
	ID   string `db:"id"`
	Name string `db:"name"`
}

func TestGetTableName(t *testing.T) {
	type tableStruct struct {
		ID string `db:"id" table:"test_table"`
	}

	table, ok := metadata.GetTableName(tableStruct{})

	if !ok || table != "test_table" {
		t.Errorf("expected table name 'test_table', got '%s' (ok=%v)", table, ok)
	}
}

func TestInsertObjectQueryBuilder_BuildFrom(t *testing.T) {
	builder := &orm.InsertObjectQueryBuilder{}
	obj := testStruct{ID: "123", Name: "Alice"}
	_, _, err := builder.BuildFrom(obj)
	if err == nil {
		t.Error("expected error due to missing table name, got nil")
	}
}
