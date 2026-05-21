package unit

import (
	"testing"
	"github.com/a-digi/coco-orm/orm/metadata"
)

type dtStruct struct {
	ID   string `db:"id"`
	Name string `db:"name"`
}

func TestGetDataTypeInfo(t *testing.T) {
	info := metadata.GetDataTypeInfo(dtStruct{})
	if len(info.Columns) != 2 {
		t.Errorf("expected 2 columns, got %d", len(info.Columns))
	}
}
