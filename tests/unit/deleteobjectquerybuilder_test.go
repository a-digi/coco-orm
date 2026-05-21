package unit

import (
	"strings"
	"testing"

	"github.com/a-digi/coco-orm/orm/orm"
)

type userDeleteStruct struct {
	ID   string `db:"id" dbtype:"UUID" table:"users"`
	Name string `db:"name"`
}

func TestDeleteObjectQueryBuilder_BuildFrom(t *testing.T) {
	builder := &orm.DeleteObjectQueryBuilder{}

	obj := userDeleteStruct{
		ID:   "123e4567-e89b-12d3-a456-426614174000",
		Name: "Alice",
	}

	identity := orm.IdentityBag{
		"id": "123e4567-e89b-12d3-a456-426614174000",
	}

	sqlStr, args, err := builder.BuildFrom(obj, identity)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(sqlStr, "DELETE FROM users") {
		t.Errorf("unexpected query start: %s", sqlStr)
	}

	if !strings.Contains(sqlStr, "WHERE id = ?") {
		t.Errorf("expected where clause not found in query: %s", sqlStr)
	}

	if len(args) != 1 {
		t.Errorf("expected 1 argument, got %d", len(args))
	}
}

func TestDeleteObjectQueryBuilder_EmptyIdentityBag(t *testing.T) {
	builder := &orm.DeleteObjectQueryBuilder{}

	obj := userDeleteStruct{}
	identity := orm.IdentityBag{}

	_, _, err := builder.BuildFrom(obj, identity)
	if err == nil {
		t.Fatal("expected error when identity bag is empty, got nil")
	}
	if !strings.Contains(err.Error(), "identity bag must be provided") {
		t.Errorf("unexpected error message: %v", err)
	}
}
