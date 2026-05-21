package orm

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/a-digi/coco-orm/orm/model"
	"github.com/a-digi/coco-orm/orm/orm"
	query_builder "github.com/a-digi/coco-orm/orm/querybuilder"
	"github.com/google/uuid"
)

const (
	DefaultPage  = 1
	DefaultLimit = 10
)

type DatabaseManager struct {
	Connector        *Connector
	MigrationFolders []string
	Hydrator         *Hydrator
}

func NewDatabaseManager(dbName string, dbDir string, migrationFolders []string) (*DatabaseManager, error) {
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	dbConfig := NewDatabaseConfig(dbName, dbDir)
	connector, err := NewConnector(dbConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create database connector: %w", err)
	}

	return &DatabaseManager{
		Connector:        connector,
		MigrationFolders: migrationFolders,
		Hydrator:         &Hydrator{},
	}, nil
}

func (manager *DatabaseManager) TableExists(tableName string) (bool, error) {
	if manager == nil || manager.Connector == nil || manager.Connector.DB == nil {
		return false, fmt.Errorf("DatabaseManager or Connector is nil")
	}

	query := `SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?;`
	var count int
	err := manager.Connector.DB.QueryRow(query, tableName).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if table exists: %w", err)
	}

	return count > 0, nil
}

func (manager *DatabaseManager) SyncMigrations() error {
	if manager == nil || manager.Connector == nil || manager.Connector.DB == nil {
		return fmt.Errorf("DatabaseManager or Connector is nil")
	}

	createTableSQL := `CREATE TABLE IF NOT EXISTS migrations (
		id TEXT PRIMARY KEY,
		created_at DATETIME NOT NULL
	);`
	_, err := manager.Connector.DB.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create 'migrations' table: %w", err)
	}

	for _, folder := range manager.MigrationFolders {
		files, err := os.ReadDir(folder)
		if err != nil {
			return fmt.Errorf("failed to read migration folder %s: %w", folder, err)
		}
		for _, file := range files {
			if file.IsDir() || filepath.Ext(file.Name()) != ".sql" {
				continue
			}
			migrationID := file.Name()
			var count int
			row := manager.Connector.DB.QueryRow("SELECT count(*) FROM migrations WHERE id = ?", migrationID)
			if err := row.Scan(&count); err != nil {
				return fmt.Errorf("failed to check migration %s: %w", migrationID, err)
			}
			if count > 0 {
				continue
			}
			migrationPath := filepath.Join(folder, file.Name())
			if err := executeSQLFile(manager.Connector.DB, migrationPath); err != nil {
				return fmt.Errorf("failed to execute migration %s: %w", migrationID, err)
			}
			_, err := manager.Connector.DB.Exec(
				"INSERT INTO migrations (id, created_at) VALUES (?, ?)",
				migrationID, time.Now().UTC().Format("2006-01-02 15:04:05"),
			)
			if err != nil {
				return fmt.Errorf("failed to record migration %s: %w", migrationID, err)
			}
		}
	}

	return nil
}

func (manager *DatabaseManager) Insert(query string, args ...interface{}) (sql.Result, error) {
	if manager == nil || manager.Connector == nil || manager.Connector.DB == nil {
		return nil, fmt.Errorf("DatabaseManager or Connector is nil")
	}

	return manager.Connector.DB.Exec(query, args...)
}

func (manager *DatabaseManager) GenerateUuidV4() string {
	return uuid.New().String()
}

func (manager *DatabaseManager) QueryBuilder() *query_builder.QueryBuilder {
	return query_builder.NewQueryBuilder()
}

func FindAll[T any](manager *DatabaseManager, entities *[]T) error {
	destType := reflect.TypeOf(entities)

	if destType.Kind() != reflect.Ptr || destType.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("FindAll: entities must be a pointer to a slice")
	}

	elemType := destType.Elem().Elem()
	if elemType.Kind() != reflect.Struct {
		return fmt.Errorf("FindAll: slice element type must be a struct, got %s", elemType.Kind())
	}

	query, err := orm.SelectQueryByObject(entities)
	if err != nil {
		return fmt.Errorf("FindAll: failed to build select query: %w", err)
	}

	rows, err := manager.Connector.DB.Query(query)
	if err != nil {
		return fmt.Errorf("FindAll: failed to execute query: %w", err)
	}
	defer rows.Close()

	return manager.Hydrator.HydrateRows(rows, entities)
}

func FindByEntity(manager *DatabaseManager, entities interface{}) error {
	destType := reflect.TypeOf(entities)
	if destType.Kind() != reflect.Ptr || destType.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("FindAllDynamic: entities must be a pointer to a slice")
	}

	elemType := destType.Elem().Elem()
	if elemType.Kind() != reflect.Struct {
		return fmt.Errorf("FindAllDynamic: slice element type must be a struct, got %s", elemType.Kind())
	}

	query, err := orm.SelectQueryByObject(entities)
	if err != nil {
		return fmt.Errorf("FindAllDynamic: failed to build select query: %w", err)
	}

	rows, err := manager.Connector.DB.Query(query)

	if err != nil {
		return fmt.Errorf("FindAllDynamic: failed to execute query: %w", err)
	}
	defer rows.Close()

	return manager.Hydrator.HydrateRows(rows, entities)
}

func FindByQuery(manager *DatabaseManager, selectQuery model.SelectQuery) error {
	if selectQuery.Entity == nil {
		return fmt.Errorf("FindByQuery: Entity is obligatory")
	}

	destType := reflect.TypeOf(selectQuery.Entity)
	if destType.Kind() != reflect.Ptr || destType.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("FindByQuery: Entity must be a pointer to a slice")
	}

	elemType := destType.Elem().Elem()
	if elemType.Kind() != reflect.Struct {
		return fmt.Errorf("FindByQuery: slice element type must be a struct, got %s", elemType.Kind())
	}

	qb, err := orm.BuildQueryBuilderFromObject(selectQuery.Entity)
	if err != nil {
		return fmt.Errorf("FindByQuery: failed to build query builder: %w", err)
	}

	for _, filter := range selectQuery.Filters {
		switch filter.Type {
		case model.FilterTypePartialMatch:
			qb.WhereLike(filter.Column, fmt.Sprintf("%%%v%%", filter.Value))
		case model.FilterTypeDateGte:
			if tm, ok := filter.Value.(time.Time); ok {
				qb.WhereDateGte(filter.Column, tm)
			} else if str, ok := filter.Value.(string); ok {
				if t, err := time.Parse(time.RFC3339, str); err == nil {
					qb.WhereDateGte(filter.Column, t)
				} else if t, err := time.Parse(time.DateOnly, str); err == nil {
					qb.WhereDateGte(filter.Column, t.Add(time.Minute))
				}
			}
		case model.FilterTypeDateLte:
			if tm, ok := filter.Value.(time.Time); ok {
				qb.WhereDateLte(filter.Column, tm)
			} else if str, ok := filter.Value.(string); ok {
				if t, err := time.Parse(time.RFC3339, str); err == nil {
					qb.WhereDateLte(filter.Column, t)
				} else if t, err := time.Parse(time.DateOnly, str); err == nil {
					qb.WhereDateLte(filter.Column, t.Add(time.Second))
				}
			}
		case model.FilterTypeGte:
			qb.WhereGte(filter.Column, filter.Value)
		case model.FilterTypeLte:
			qb.WhereLte(filter.Column, filter.Value)
		case model.FilterTypeExactMatch:
			fallthrough
		default:
			qb.Where(fmt.Sprintf("%s = ?", filter.Column), filter.Value)
		}
	}

	for _, sort := range selectQuery.Sorting {
		dir := string(sort.Direction)
		if dir == "" {
			dir = string(model.SortDirectionAsc)
		}
		qb.OrderBy(fmt.Sprintf("%s %s", sort.Column, dir))
	}

	limit := selectQuery.Pagination.Limit
	if limit > 0 {
		qb.Limit(limit)
	} else if limit == 0 {
		limit = DefaultLimit
		qb.Limit(limit)
	}

	if selectQuery.Pagination.Page > 1 {
		qb.Offset((selectQuery.Pagination.Page - 1) * limit)
	} else if selectQuery.Pagination.Page == 0 {
		qb.Offset(0)
	}

	query := qb.Build()

	rows, err := manager.Connector.DB.Query(query.SQL, query.Args...)
	if err != nil {
		return fmt.Errorf("FindByQuery: failed to execute query: %w", err)
	}
	defer rows.Close()

	return manager.Hydrator.HydrateRows(rows, selectQuery.Entity)
}

func executeSQLFile(db *sql.DB, path string) error {
	content, err := os.ReadFile(path)

	if err != nil {
		return fmt.Errorf("failed to read SQL file: %w", err)
	}

	statements := strings.Split(string(content), ";")

	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("failed to execute statement: %w", err)
		}
	}

	return nil
}
