package orm

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/a-digi/coco-orm/orm/metadata"
	query_builder "github.com/a-digi/coco-orm/orm/querybuilder"
)

type DeleteObjectQueryBuilder struct {
	cache sync.Map
}

func (b *DeleteObjectQueryBuilder) ExtractDeleteMeta(obj interface{}) (string, error) {
	_, _, tableName, _, err := metadata.ExtractTypeMeta(
		obj,
		func(t reflect.Type) (metadata.CacheEntry, bool) {
			entry, ok := b.cache.Load(t)
			if !ok {
				return metadata.CacheEntry{}, false
			}
			return entry.(metadata.CacheEntry), true
		},
		func(t reflect.Type, entry metadata.CacheEntry) {
			b.cache.Store(t, entry)
		},
	)
	if err != nil {
		return "", err
	}

	return tableName, nil
}

// BuildFrom generates a DELETE query.
// It requires an IdentityBag (blueprint) to identify the specific entries to delete and prevent catastrophic entire-table deletions.
func (b *DeleteObjectQueryBuilder) BuildFrom(obj interface{}, identity IdentityBag) (string, []interface{}, error) {
	if len(identity) == 0 {
		return "", nil, fmt.Errorf("an identity bag must be provided to block unregulated deletions across the database table")
	}

	tableName, err := b.ExtractDeleteMeta(obj)
	if err != nil {
		return "", nil, err
	}

	builder := query_builder.NewDeleteQueryBuilder()
	builder.From(tableName)

	for col, val := range identity {
		builder.Where(fmt.Sprintf("%s = ?", col), val)
	}

	query, err := builder.Build()
	if err != nil {
		return "", nil, err
	}

	return query.SQL, query.Args, nil
}
