package unit

import (
	"testing"

	"github.com/a-digi/coco-orm/orm/querybuilder"
)

func TestDeleteQueryBuilder_Build(t *testing.T) {
	b := querybuilder.NewDeleteQueryBuilder()

	_, err := b.Build()
	if err == nil {
		t.Fatal("expected error for empty table, got nil")
	}

	b.From("users").Where("id = ?", 1).OrWhere("role = ?", "admin")

	query, err := b.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedSQL := "DELETE FROM users WHERE (id = ?) OR (role = ?)"
	if query.SQL != expectedSQL {
		t.Errorf("unexpected query: %s", query.SQL)
	}

	if len(query.Args) != 2 || query.Args[0] != 1 || query.Args[1] != "admin" {
		t.Errorf("unexpected args: %v", query.Args)
	}

	// Test Limit and Offset
	b.Limit(5).Offset(10)
	query2, err := b.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedSQL2 := "DELETE FROM users WHERE (id = ?) OR (role = ?) LIMIT 5 OFFSET 10"
	if query2.SQL != expectedSQL2 {
		t.Errorf("unexpected query with limit/offset: %s", query2.SQL)
	}
}
