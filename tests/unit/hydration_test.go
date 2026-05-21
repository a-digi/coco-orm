package unit

import (
	"database/sql"
	"testing"
	"github.com/a-digi/coco-orm/orm"
)

type hydrationStruct struct {
	ID   int
	Name string
}

func TestHydrator_HydrateRows_InvalidDest(t *testing.T) {
	h := &orm.Hydrator{}
	rows := &sql.Rows{} // Dummy, wird nicht genutzt
	err := h.HydrateRows(rows, []hydrationStruct{})
	if err == nil {
		t.Error("expected error for non-pointer slice")
	}
}
