package unit

import (
	"testing"

	"github.com/a-digi/coco-orm/orm/querybuilder"
)

func TestUpdateQueryBuilder_Build(t *testing.T) {
	b := &querybuilder.UpdateQueryBuilder{}
	b.Table("users").Columns("name", "age").Values("Max", 30).Where("id = ?", 1)
	query, args, err := b.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedQuery := "UPDATE users SET name = ?, age = ? WHERE id = ?"
	if query != expectedQuery {
		t.Errorf("unexpected query: %s", query)
	}
	if len(args) != 3 {
		t.Errorf("unexpected args length: %v", args)
	}
	if args[0] != "Max" || args[1] != 30 || args[2] != 1 {
		t.Errorf("unexpected args: %v", args)
	}

	// Test Set method
	b2 := &querybuilder.UpdateQueryBuilder{}
	b2.Table("users").Set("status", "active").Where("id = ?", 5)
	query2, args2, err := b2.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedQuery2 := "UPDATE users SET status = ? WHERE id = ?"
	if query2 != expectedQuery2 {
		t.Errorf("unexpected query: %s", query2)
	}
	if len(args2) != 2 || args2[0] != "active" || args2[1] != 5 {
		t.Errorf("unexpected args: %v", args2)
	}
}
