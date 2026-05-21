package tests

import (
	"os"
	"testing"
	"time"

	"github.com/a-digi/coco-orm/orm"
)

type M2MGroup struct {
	_         struct{} `table:"groups"`
	ID        string   `db:"id" dbtype:"UUID" nullable:"false" json:"id"`
	Name      string   `db:"name" dbtype:"TEXT" nullable:"false" json:"name"`
	CreatedAt string   `db:"created_at" dbtype:"DATETIME" nullable:"true" json:"created_at"`
}

type M2MUser struct {
	_         struct{}   `table:"users"`
	ID        string     `db:"id" dbtype:"UUID" nullable:"false" json:"id"`
	Username  string     `db:"username" dbtype:"TEXT" nullable:"false" json:"username"`
	CreatedAt string     `db:"created_at" dbtype:"DATETIME" nullable:"true" json:"created_at"`
	Groups    []M2MGroup `relation:"m2m" join_entity:"M2MUserGroupMember" join_table:"user_group_members" join_fk:"user_id" join_ass_fk:"group_id" json:"groups,omitempty"`
}

type M2MUserGroupMember struct {
	_        struct{} `table:"user_group_members"`
	ID       string   `db:"id" dbtype:"UUID" nullable:"false" json:"id"`
	GroupID  string   `db:"group_id" dbtype:"TEXT" nullable:"false" json:"group_id"`
	UserID   string   `db:"user_id" dbtype:"TEXT" nullable:"false" json:"user_id"`
	IsActive bool     `db:"is_active" dbtype:"BOOLEAN" nullable:"false" default:"true" json:"is_active"`
}

func setupTestDB() (*orm.DatabaseManager, string, error) {
	dbName := "test_m2m.db"
	dbDir := "./"

	// Create required folders
	os.RemoveAll("migrations")
	os.MkdirAll("migrations", 0755)

	migrationContent := `
CREATE TABLE users (
	id TEXT PRIMARY KEY,
	username TEXT NOT NULL,
	created_at DATETIME
);

CREATE TABLE groups (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	created_at DATETIME
);

CREATE TABLE user_group_members (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	group_id TEXT NOT NULL,
	is_active BOOLEAN
);
`
	os.WriteFile("migrations/001_init.sql", []byte(migrationContent), 0644)

	manager, err := orm.NewDatabaseManager(dbName, dbDir, []string{"migrations"})
	if err != nil {
		return nil, "", err
	}

	err = manager.SyncMigrations()
	if err != nil {
		return nil, "", err
	}

	return manager, dbName, nil
}

func teardownTestDB(dbName string) {
	os.Remove(dbName)
	os.RemoveAll("migrations")
}

func TestManyToManyGeneric(t *testing.T) {
	manager, dbName, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup DB: %v", err)
	}
	defer teardownTestDB(dbName)

	userID := manager.GenerateUuidV4()
	groupID1 := manager.GenerateUuidV4()
	groupID2 := manager.GenerateUuidV4()

	// Insert User
	_, err = manager.Connector.DB.Exec("INSERT INTO users (id, username, created_at) VALUES (?, ?, ?)", userID, "testuser", time.Now().Format("2006-01-02 15:04:05"))
	if err != nil {
		t.Fatalf("Failed to insert user: %v", err)
	}

	// Insert Groups
	_, err = manager.Connector.DB.Exec("INSERT INTO groups (id, name, created_at) VALUES (?, ?, ?)", groupID1, "Group 1", time.Now().Format("2006-01-02 15:04:05"))
	if err != nil {
		t.Fatalf("Failed to insert group 1: %v", err)
	}
	_, err = manager.Connector.DB.Exec("INSERT INTO groups (id, name, created_at) VALUES (?, ?, ?)", groupID2, "Group 2", time.Now().Format("2006-01-02 15:04:05"))
	if err != nil {
		t.Fatalf("Failed to insert group 2: %v", err)
	}

	// Setup relations into standard struct to test SaveManyToMany
	user := M2MUser{
		ID:       userID,
		Username: "testuser",
		Groups: []M2MGroup{
			{ID: groupID1},
			{ID: groupID2},
		},
	}

	// Test Save
	err = orm.SaveManyToMany[M2MUser, M2MGroup, M2MUserGroupMember](manager, &user, "Groups")
	if err != nil {
		t.Fatalf("Failed to run SaveManyToMany: %v", err)
	}

	// Verify pivot table hydrated the explicit Junction entity values like generated UUIDs
	rows, err := manager.Connector.DB.Query("SELECT id, user_id, group_id, is_active FROM user_group_members WHERE user_id = ?", userID)
	if err != nil {
		t.Fatalf("Failed to query pivot table manually: %v", err)
	}
	defer rows.Close()

	var count int
	for rows.Next() {
		count++
		var jId, uId, gId string
		var isAct bool
		err := rows.Scan(&jId, &uId, &gId, &isAct)
		if err != nil {
			t.Fatalf("Failed to scan pivot record: %v", err)
		}
		if len(jId) < 10 { // naive UUID length check ensures UUID generator triggered
			t.Errorf("Expected valid UUID in junction table, got: %s", jId)
		}
		if uId != userID {
			t.Errorf("Expected userID %s, got %s", userID, uId)
		}
		if isAct != true {
			t.Errorf("Expected default true for is_active in junction table")
		}
	}

	if count != 2 {
		t.Errorf("Expected 2 pivot records, found %d", count)
	}

	// Note: Currently ORM doesn't have a single-entity fetch. So we use FindAll
	var users []M2MUser
	err = orm.FindAll(manager, &users)
	if err != nil {
		t.Fatalf("Failed to FindAll: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("Expected 1 user, found %d", len(users))
	}

	// Test Eager Loading
	err = orm.EagerLoadManyToMany[M2MUser, M2MGroup](manager, &users, "Groups")
	if err != nil {
		t.Fatalf("Failed to EagerLoadManyToMany: %v", err)
	}

	if len(users[0].Groups) != 2 {
		t.Errorf("Expected eager loaded 2 groups, got %d", len(users[0].Groups))
	}

	// Ensure mapped group details exist
	foundG1 := false
	for _, g := range users[0].Groups {
		if g.ID == groupID1 && g.Name == "Group 1" {
			foundG1 = true
		}
	}

	if !foundG1 {
		t.Errorf("Failed to properly hydrate relation targets into the user struct")
	}
}
