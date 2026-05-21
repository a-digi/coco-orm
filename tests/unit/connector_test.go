package unit

import (
	"os"
	"testing"
	"github.com/a-digi/coco-orm/orm"
)

func TestNewConnector(t *testing.T) {
	dir := "./testdb"
	defer os.RemoveAll(dir)
	cfg := orm.DatabaseConfig{DbName: "test.db", DirectoryPath: dir}
	_, err := orm.NewConnector(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
