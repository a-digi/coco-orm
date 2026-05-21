package unit

import (
	"strings"
	"testing"

	"github.com/a-digi/coco-orm/orm/orm"
)

type userUpdateStruct struct {
	ID     string `db:"id" dbtype:"UUID" table:"users"`
	Name   string `db:"name"`
	Active bool   `db:"is_active" dbtype:"BOOLEAN"`
}

func TestUpdateObjectQueryBuilder_BuildFrom(t *testing.T) {
	builder := &orm.UpdateObjectQueryBuilder{}

	obj := userUpdateStruct{
		ID:     "123e4567-e89b-12d3-a456-426614174000",
		Name:   "Alice",
		Active: true,
	}

	identity := orm.IdentityBag{
		"id": "123e4567-e89b-12d3-a456-426614174000",
	}

	sqlStr, args, err := builder.BuildFrom(obj, identity)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(sqlStr, "UPDATE users SET") {
		t.Errorf("unexpected query: %s", sqlStr)
	}

	if !strings.Contains(sqlStr, "WHERE id = ?") {
		t.Errorf("expected where clause not found in query: %s", sqlStr)
	}

	// We expect 4 args total: 3 for SET and 1 for WHERE
	if len(args) != 4 {
		t.Errorf("expected 4 arguments, got %d", len(args))
	}
}

func TestUpdateObjectQueryBuilder_EmptyIdentityBag(t *testing.T) {
	builder := &orm.UpdateObjectQueryBuilder{}

	obj := userUpdateStruct{
		Name: "Alice",
	}

	identity := orm.IdentityBag{}

	_, _, err := builder.BuildFrom(obj, identity)
	if err == nil {
		t.Fatal("expected error when identity bag is empty, got nil")
	}
	if !strings.Contains(err.Error(), "identity bag must be provided") {
		t.Errorf("unexpected error message: %v", err)
	}
}
