package unit

import (
	"os"
	"testing"
	"github.com/a-digi/coco-orm/orm"
)

func TestNewDatabaseManager(t *testing.T) {
	dbName := "test.db"
	dbDir := "./testdb"
	migrationFolders := []string{}
	defer os.RemoveAll(dbDir)
	_, err := orm.NewDatabaseManager(dbName, dbDir, migrationFolders)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
