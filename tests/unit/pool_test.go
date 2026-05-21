package unit

import (
	"sync"
	"testing"
	"time"

	"github.com/a-digi/coco-orm/orm/pool"
)

// emptyMigrations returns a temp dir that exists but has no SQL files,
// so SyncMigrations just creates the migrations table and succeeds.
func emptyMigrations(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

func TestPool_GetReturnsSameManager(t *testing.T) {
	p := pool.New(pool.Config{})
	defer p.Close()

	dir := t.TempDir()
	mig := emptyMigrations(t)

	mgr1, err := p.Get("test.db", dir, []string{mig})
	if err != nil {
		t.Fatalf("first Get: %v", err)
	}
	mgr2, err := p.Get("test.db", dir, []string{mig})
	if err != nil {
		t.Fatalf("second Get: %v", err)
	}
	if mgr1 != mgr2 {
		t.Error("expected same *DatabaseManager on repeated Get")
	}
}

func TestPool_MigrationsRunOnOpen(t *testing.T) {
	p := pool.New(pool.Config{})
	defer p.Close()

	dir := t.TempDir()
	mig := emptyMigrations(t)

	mgr, err := p.Get("test.db", dir, []string{mig})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	// SyncMigrations creates the `migrations` table — verify it exists.
	exists, err := mgr.TableExists("migrations")
	if err != nil {
		t.Fatalf("TableExists: %v", err)
	}
	if !exists {
		t.Error("expected migrations table to exist after pool.Get")
	}
}

func TestPool_LRUEvictsWhenFull(t *testing.T) {
	p := pool.New(pool.Config{Capacity: 2})
	defer p.Close()

	mig := emptyMigrations(t)
	dir1, dir2, dir3 := t.TempDir(), t.TempDir(), t.TempDir()

	if _, err := p.Get("a.db", dir1, []string{mig}); err != nil {
		t.Fatalf("Get dir1: %v", err)
	}
	if _, err := p.Get("b.db", dir2, []string{mig}); err != nil {
		t.Fatalf("Get dir2: %v", err)
	}
	// dir1 was MRU at add time, dir2 is now MRU — adding dir3 evicts dir1 (LRU).
	// But first, touch dir2 again so dir1 is definitely LRU.
	if _, err := p.Get("b.db", dir2, []string{mig}); err != nil {
		t.Fatalf("re-Get dir2: %v", err)
	}
	if _, err := p.Get("c.db", dir3, []string{mig}); err != nil {
		t.Fatalf("Get dir3: %v", err)
	}

	keys := p.Keys()
	if len(keys) != 2 {
		t.Errorf("expected 2 keys after LRU eviction, got %d: %v", len(keys), keys)
	}
	for _, k := range keys {
		if k == dir1+"/a.db" {
			t.Error("LRU entry (dir1/a.db) should have been evicted")
		}
	}
}

func TestPool_EvictIdle(t *testing.T) {
	p := pool.New(pool.Config{IdleTTL: 50 * time.Millisecond})
	defer p.Close()

	mig := emptyMigrations(t)
	dir := t.TempDir()

	if _, err := p.Get("idle.db", dir, []string{mig}); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if p.Stats().Size != 1 {
		t.Fatal("expected 1 entry before idle eviction")
	}

	time.Sleep(100 * time.Millisecond)
	p.EvictIdle()

	if p.Stats().Size != 0 {
		t.Errorf("expected 0 entries after idle eviction, got %d", p.Stats().Size)
	}
}

func TestPool_EvictRemovesEntry(t *testing.T) {
	p := pool.New(pool.Config{})
	defer p.Close()

	mig := emptyMigrations(t)
	dir := t.TempDir()

	mgr1, err := p.Get("evict.db", dir, []string{mig})
	if err != nil {
		t.Fatalf("first Get: %v", err)
	}

	p.Evict("evict.db", dir)

	if p.Stats().Size != 0 {
		t.Error("expected empty pool after Evict")
	}

	mgr2, err := p.Get("evict.db", dir, []string{mig})
	if err != nil {
		t.Fatalf("Get after evict: %v", err)
	}
	if mgr1 == mgr2 {
		t.Error("expected a fresh *DatabaseManager after evict+re-get")
	}
}

func TestPool_StatsCountsCorrectly(t *testing.T) {
	p := pool.New(pool.Config{Capacity: 10})
	defer p.Close()

	mig := emptyMigrations(t)
	dir1, dir2 := t.TempDir(), t.TempDir()

	if _, err := p.Get("x.db", dir1, []string{mig}); err != nil {
		t.Fatalf("Get dir1: %v", err)
	}
	if _, err := p.Get("x.db", dir1, []string{mig}); err != nil {
		t.Fatalf("re-Get dir1: %v", err)
	}
	if _, err := p.Get("y.db", dir2, []string{mig}); err != nil {
		t.Fatalf("Get dir2: %v", err)
	}

	s := p.Stats()
	if s.Size != 2 {
		t.Errorf("Stats.Size: want 2, got %d", s.Size)
	}
	if s.Capacity != 10 {
		t.Errorf("Stats.Capacity: want 10, got %d", s.Capacity)
	}
	for _, e := range s.Entries {
		if e.Key == dir1+"/x.db" && e.Hits != 2 {
			t.Errorf("Hits for dir1/x.db: want 2, got %d", e.Hits)
		}
	}
}

func TestPool_ConcurrentGets(t *testing.T) {
	p := pool.New(pool.Config{})
	defer p.Close()

	mig := emptyMigrations(t)
	dir := t.TempDir()

	const goroutines = 20
	results := make([]*struct{ mgr interface{} }, goroutines)
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			mgr, err := p.Get("concurrent.db", dir, []string{mig})
			if err != nil {
				t.Errorf("goroutine %d Get: %v", i, err)
				return
			}
			results[i] = &struct{ mgr interface{} }{mgr: mgr}
		}()
	}
	wg.Wait()

	// All goroutines must have received the same manager pointer.
	var first interface{}
	for i, r := range results {
		if r == nil {
			continue
		}
		if first == nil {
			first = r.mgr
		} else if r.mgr != first {
			t.Errorf("goroutine %d got a different *DatabaseManager", i)
		}
	}
}

func TestPool_CloseEmptiesPool(t *testing.T) {
	p := pool.New(pool.Config{})

	mig := emptyMigrations(t)
	dir1, dir2 := t.TempDir(), t.TempDir()

	if _, err := p.Get("a.db", dir1, []string{mig}); err != nil {
		t.Fatalf("Get dir1: %v", err)
	}
	if _, err := p.Get("b.db", dir2, []string{mig}); err != nil {
		t.Fatalf("Get dir2: %v", err)
	}
	if p.Stats().Size != 2 {
		t.Fatal("expected 2 entries before Close")
	}

	p.Close()

	if p.Stats().Size != 0 {
		t.Errorf("expected 0 entries after Close, got %d", p.Stats().Size)
	}
	if len(p.Keys()) != 0 {
		t.Error("expected empty Keys() after Close")
	}
}
