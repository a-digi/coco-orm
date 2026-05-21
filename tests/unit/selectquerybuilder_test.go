package unit

import (
	"testing"
	"github.com/a-digi/coco-orm/orm/querybuilder"
)

func TestQueryBuilder_Build(t *testing.T) {
	b := querybuilder.NewQueryBuilder()
	b.Select("id", "name").From("users").Where("id = ?", 1)
	q := b.Build()
	if q.SQL == "" || len(q.Args) != 1 {
		t.Errorf("unexpected query or args: %s, %v", q.SQL, q.Args)
	}
}
