package unit

import (
	"testing"
	"github.com/a-digi/coco-orm/orm/metadata"
)

type tableTestStruct struct {
	ID string `db:"id" table:"my_table"`
}

func TestGetTableName_TableTag(t *testing.T) {
	table, ok := metadata.GetTableName(tableTestStruct{})
	if !ok || table != "my_table" {
		t.Errorf("expected table name 'my_table', got '%s' (ok=%v)", table, ok)
	}
}

func TestGetTableName_NoTag(t *testing.T) {
	type noTagStruct struct {
		ID string
	}
	table, ok := metadata.GetTableName(noTagStruct{})
	if ok || table != "" {
		t.Errorf("expected no table name, got '%s' (ok=%v)", table, ok)
	}
}
