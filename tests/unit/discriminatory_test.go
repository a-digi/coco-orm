package unit

import (
	"testing"
	"github.com/a-digi/coco-orm/orm/metadata"
	"github.com/a-digi/coco-orm/orm/orm"
)

type adminGroup struct {
	_                struct{} `table:"user_groups"`
	ID               string   `db:"id" dbtype:"UUID" nullable:"false" json:"id"`
	GroupType        string   `db:"group_type" dbtype:"TEXT" nullable:"false" discriminatory:"admin" default:"admin" json:"group_type"`
	Title            string   `db:"title" dbtype:"TEXT" nullable:"false" json:"title"`
	GroupDescription string   `db:"group_description" dbtype:"TEXT" nullable:"true" json:"group_description"`
	CreatedAt        string   `db:"created_at" dbtype:"DATETIME" nullable:"true" json:"created_at"`
	IsActive         bool     `db:"is_active" dbtype:"BOOLEAN" nullable:"false" default:"true" json:"is_active"`
}

func TestDiscriminatoryTagInMetadata(t *testing.T) {
	info := metadata.GetDataTypeInfo(adminGroup{})
	if info.DiscriminatoryCol != "group_type" {
		t.Errorf("expected discriminatory column 'group_type', got '%s'", info.DiscriminatoryCol)
	}
	if info.DiscriminatoryVal != "admin" {
		t.Errorf("expected discriminatory value 'admin', got '%s'", info.DiscriminatoryVal)
	}
}

func TestDiscriminatoryWhereInQueryBuilder(t *testing.T) {
	var groups []adminGroup
	qb, err := orm.BuildQueryBuilderFromObject(&groups)
	if err != nil {
		t.Fatalf("BuildQueryBuilderFromObject failed: %v", err)
	}
	query := qb.Build()
	if query.SQL == "" {
		t.Error("expected non-empty SQL query")
	}
	if !(len(query.Args) > 0 && query.Args[0] == "admin") {
		t.Errorf("expected discriminatory value 'admin' in query args, got %v", query.Args)
	}
	if query.SQL == "" || query.Args[0] != "admin" || !containsWhereOnGroupType(query.SQL) {
		t.Errorf("expected WHERE on group_type in SQL, got: %s", query.SQL)
	}
}

func containsWhereOnGroupType(sql string) bool {
	return (len(sql) > 0 && (sql == "SELECT id, group_type, title, group_description, created_at, is_active FROM user_groups WHERE group_type = ?" ||
		(sql[:44] == "SELECT id, group_type, title, group_description, created_at, is_active FROM user_groups WHERE group_type = ?")))
}

