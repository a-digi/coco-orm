package unit

import (
	"os"
	"testing"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/a-digi/coco-orm/orm"
)

type userTestStruct struct {
	ID   int    `db:"id" table:"users"`
	Name string `db:"name"`
}

func setupTestDB(t *testing.T) (*orm.DatabaseManager, func()) {
	dbFile := "./test_findall.db"
	os.Remove(dbFile)
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	_, err = db.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT);`)
	if err != nil {
		db.Close(); os.Remove(dbFile)
		t.Fatalf("failed to create table: %v", err)
	}
	_, err = db.Exec(`INSERT INTO users (id, name) VALUES (1, 'Alice'), (2, 'Bob');`)
	if err != nil {
		db.Close(); os.Remove(dbFile)
		t.Fatalf("failed to insert test data: %v", err)
	}
	manager := &orm.DatabaseManager{
		Connector: &orm.Connector{DB: db},
		Hydrator:  &orm.Hydrator{},
	}
	cleanup := func() { db.Close(); os.Remove(dbFile) }
	return manager, cleanup
}

func TestDatabaseManager_FindAll(t *testing.T) {
	manager, cleanup := setupTestDB(t)
	defer cleanup()

	var users []userTestStruct
	err := orm.FindAll(manager, &users)
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
	if users[0].Name != "Alice" || users[1].Name != "Bob" {
		t.Errorf("unexpected user data: %+v", users)
	}
}

func TestDatabaseManager_FindAll_NoTableTag(t *testing.T) {
	manager, cleanup := setupTestDB(t)
	defer cleanup()
	type noTableStruct struct {
		ID int `db:"id"`
	}
	var items []noTableStruct
	err := orm.FindAll(manager, &items)
	if err == nil {
		t.Error("expected error for missing table tag, got nil")
	}
}
