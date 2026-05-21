package unit

import (
	"testing"
	"github.com/a-digi/coco-orm/orm/querybuilder"
)

func TestInsertQueryBuilder_Build(t *testing.T) {
	b := &querybuilder.InsertQueryBuilder{}
	b.Into("users").Columns("id", "name").Values(1, "Max")
	query, args, err := b.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if query == "" || len(args) != 2 {
		t.Errorf("unexpected query or args: %s, %v", query, args)
	}
}
