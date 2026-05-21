package unit

import (
	"testing"
	"github.com/a-digi/coco-orm/orm/orm"
)

type defaultStruct struct {
	ID    int    `db:"id"`
	Name  string `db:"name" default:"TestName"`
	Count int    `db:"count" default:"42"`
}

func TestInsertObjectQueryBuilder_DefaultValue(t *testing.T) {
	obj := defaultStruct{ID: 1}
	builder := &orm.InsertObjectQueryBuilder{}
	table, columns, values, err := builder.ExtractInsertMetaAndValues(obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if table == "" {
		t.Error("expected table name, got empty string")
	}
	foundName, foundCount := false, false
	for i, col := range columns {
		if col == "name" && values[i] != "TestName" {
			t.Errorf("expected default value 'TestName' for name, got %v", values[i])
			foundName = true
		}
		if col == "count" && values[i] != 42 {
			t.Errorf("expected default value 42 for count, got %v", values[i])
			foundCount = true
		}
	}
	if !foundName {
		t.Error("column 'name' with default value not found")
	}
	if !foundCount {
		t.Error("column 'count' with default value not found")
	}
}

