package orm

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"fmt"
	"os"
	"sync"
	"path/filepath"
)

var (
	pool      = make(map[string]*sql.DB)
	poolMutex sync.Mutex
)

type Connector struct {
	DB   *sql.DB
	path string
}

type DatabaseConfig struct {
	DbName      string
	DirectoryPath string
}

func NewDatabaseConfig(dbName, directoryPath string) DatabaseConfig {
	return DatabaseConfig{
		DbName: dbName,
		DirectoryPath: directoryPath,
	}
}

func NewConnector(cfg DatabaseConfig) (*Connector, error) {
	path := filepath.Join(cfg.DirectoryPath, cfg.DbName)
	if err := os.MkdirAll(cfg.DirectoryPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}
	poolMutex.Lock()
	db, ok := pool[path]
	poolMutex.Unlock()

	if ok {
		if err := db.Ping(); err == nil {
			return &Connector{DB: db, path: path}, nil
		} else {
			poolMutex.Lock()
			delete(pool, path)
			poolMutex.Unlock()
			_ = db.Close()
		}
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("Fehler beim Öffnen der Datenbank: %w", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("Fehler beim Verbinden mit der Datenbank: %w", err)
	}
	poolMutex.Lock()
	pool[path] = db
	poolMutex.Unlock()

	return &Connector{DB: db, path: path}, nil
}

func (c *Connector) Close() error {
	if c.DB != nil {
		poolMutex.Lock()
		delete(pool, c.path)
		poolMutex.Unlock()
		return c.DB.Close()
	}
	return nil
}

func CloseAllConnections() error {
	poolMutex.Lock()
	defer poolMutex.Unlock()
	var firstErr error

	for path, db := range pool {
		err := db.Close()
		if err != nil && firstErr == nil {
			firstErr = err
		}
		delete(pool, path)
	}

	return firstErr
}
