package unit

import (
	"testing"
	"github.com/a-digi/coco-orm/orm"
)

func TestMigrationStruct(t *testing.T) {
	m := orm.Migration{}
	if m.ID != "" {
		t.Errorf("expected empty ID, got %s", m.ID)
	}
}
