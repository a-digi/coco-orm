package pool

import (
	"container/list"
	"fmt"
	"sync"
	"time"

	"github.com/a-digi/coco-orm/orm"
)

// Config controls pool capacity and connection settings.
// All fields are optional; zero values disable the respective feature.
type Config struct {
	// Capacity is the maximum number of DatabaseManagers to keep open.
	// When the pool is full, the least-recently-used entry is closed and
	// evicted before the new one is inserted. 0 means unlimited.
	Capacity int

	// IdleTTL is how long an entry can go unused before EvictIdle closes
	// it. 0 disables TTL-based eviction.
	IdleTTL time.Duration

	// MaxOpenConns is forwarded to sql.DB.SetMaxOpenConns for every new
	// connection. 0 leaves the driver default in place.
	MaxOpenConns int

	// MaxIdleConns is forwarded to sql.DB.SetMaxIdleConns for every new
	// connection. 0 leaves the driver default in place.
	MaxIdleConns int
}

// EntryStats is a point-in-time snapshot of one cached entry.
type EntryStats struct {
	Key      string
	Hits     int64
	LastUsed time.Time
}

// Stats is a point-in-time snapshot of the whole pool.
type Stats struct {
	Size     int
	Capacity int
	Entries  []EntryStats
}

type entry struct {
	key      string
	mgr      *orm.DatabaseManager
	hits     int64
	lastUsed time.Time
	elem     *list.Element
}

// Pool is a thread-safe LRU cache of orm.DatabaseManager instances keyed by
// (dbName, dir). Use New to construct one; the zero value is not usable.
type Pool struct {
	cfg   Config
	mu    sync.Mutex
	items map[string]*entry
	lru   *list.List // front = MRU, back = LRU
}

// New constructs a Pool with the given configuration.
func New(cfg Config) *Pool {
	return &Pool{
		cfg:   cfg,
		items: make(map[string]*entry),
		lru:   list.New(),
	}
}

// Get returns the DatabaseManager for (dbName, dir). On a cache miss the DB
// is opened, all migrationFolders are applied via SyncMigrations, and the
// result is cached. Subsequent calls for the same key return the cached
// manager without re-running migrations.
func (p *Pool) Get(dbName, dir string, migrationFolders []string) (*orm.DatabaseManager, error) {
	key := dir + "/" + dbName

	p.mu.Lock()
	defer p.mu.Unlock()

	if e, ok := p.items[key]; ok {
		e.hits++
		e.lastUsed = time.Now()
		p.lru.MoveToFront(e.elem)
		return e.mgr, nil
	}

	mgr, err := orm.NewDatabaseManager(dbName, dir, migrationFolders)
	if err != nil {
		return nil, fmt.Errorf("pool: open %s: %w", key, err)
	}
	if err := mgr.SyncMigrations(); err != nil {
		_ = mgr.Connector.DB.Close()
		return nil, fmt.Errorf("pool: migrate %s: %w", key, err)
	}
	if p.cfg.MaxOpenConns > 0 {
		mgr.Connector.DB.SetMaxOpenConns(p.cfg.MaxOpenConns)
	}
	if p.cfg.MaxIdleConns > 0 {
		mgr.Connector.DB.SetMaxIdleConns(p.cfg.MaxIdleConns)
	}

	e := &entry{key: key, mgr: mgr, hits: 1, lastUsed: time.Now()}
	e.elem = p.lru.PushFront(e)
	p.items[key] = e

	if p.cfg.Capacity > 0 && p.lru.Len() > p.cfg.Capacity {
		p.evictLRU()
	}

	return mgr, nil
}

// Evict closes and removes the entry for (dbName, dir). No-op if the entry
// does not exist.
func (p *Pool) Evict(dbName, dir string) {
	key := dir + "/" + dbName
	p.mu.Lock()
	defer p.mu.Unlock()
	if e, ok := p.items[key]; ok {
		p.removeEntry(e)
	}
}

// EvictIdle closes entries that have been idle longer than cfg.IdleTTL.
// No-op when cfg.IdleTTL is zero. Intended to be called from a periodic
// background goroutine.
func (p *Pool) EvictIdle() {
	if p.cfg.IdleTTL <= 0 {
		return
	}
	cutoff := time.Now().Add(-p.cfg.IdleTTL)
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, e := range p.items {
		if e.lastUsed.Before(cutoff) {
			p.removeEntry(e)
		}
	}
}

// Keys returns a snapshot of all currently cached pool keys.
func (p *Pool) Keys() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	keys := make([]string, 0, len(p.items))
	for k := range p.items {
		keys = append(keys, k)
	}
	return keys
}

// Stats returns a point-in-time snapshot of pool metrics.
func (p *Pool) Stats() Stats {
	p.mu.Lock()
	defer p.mu.Unlock()
	entries := make([]EntryStats, 0, len(p.items))
	for _, e := range p.items {
		entries = append(entries, EntryStats{
			Key:      e.key,
			Hits:     e.hits,
			LastUsed: e.lastUsed,
		})
	}
	return Stats{
		Size:     len(p.items),
		Capacity: p.cfg.Capacity,
		Entries:  entries,
	}
}

// Close shuts down every cached connection and empties the pool.
func (p *Pool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, e := range p.items {
		closeEntry(e)
	}
	p.items = make(map[string]*entry)
	p.lru.Init()
}

func (p *Pool) evictLRU() {
	back := p.lru.Back()
	if back == nil {
		return
	}
	p.removeEntry(back.Value.(*entry))
}

func (p *Pool) removeEntry(e *entry) {
	closeEntry(e)
	p.lru.Remove(e.elem)
	delete(p.items, e.key)
}

func closeEntry(e *entry) {
	if e.mgr != nil && e.mgr.Connector != nil && e.mgr.Connector.DB != nil {
		_ = e.mgr.Connector.DB.Close()
	}
}
