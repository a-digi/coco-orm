package orm

import (
	"fmt"
	"reflect"

	"github.com/a-digi/coco-orm/orm/metadata"
	"github.com/a-digi/coco-orm/orm/orm"
)

// SaveManyToMany synchronization logic.
// J represents the junction entity (e.g. UserGroupMember)
// S represents the source entity (e.g. User)
// T represents the target entity (e.g. Group)
func SaveManyToMany[S any, T any, J any](manager *DatabaseManager, source *S, relationFieldName string) error {
	if manager == nil || manager.Connector == nil || manager.Connector.DB == nil {
		return fmt.Errorf("DatabaseManager or Connector is nil")
	}

	sourceVal := reflect.ValueOf(source).Elem()
	srcIDField := sourceVal.FieldByName("ID")
	if !srcIDField.IsValid() {
		return fmt.Errorf("SaveManyToMany: source entity does not have an ID field")
	}
	srcID := fmt.Sprintf("%v", srcIDField.Interface())

	var dummyS S
	sourceMeta := metadata.GetDataTypeInfo(dummyS)
	var relInfo *metadata.RelationInfo
	for _, rel := range sourceMeta.Relations {
		if rel.FieldName == relationFieldName {
			relInfo = &rel
			break
		}
	}

	if relInfo == nil || relInfo.Type != "m2m" {
		return fmt.Errorf("SaveManyToMany: m2m relation %s not found", relationFieldName)
	}

	// Retrieve the associated T slice
	relationField := sourceVal.FieldByName(relationFieldName)
	if !relationField.IsValid() || relationField.Kind() != reflect.Slice {
		return fmt.Errorf("SaveManyToMany: relation field %s is not a slice", relationFieldName)
	}

	// Delete existing junction entries for this source
	deleteSql := fmt.Sprintf("DELETE FROM %s WHERE %s = ?", relInfo.JoinTable, relInfo.JoinFK)
	_, err := manager.Insert(deleteSql, srcID)
	if err != nil {
		return fmt.Errorf("SaveManyToMany: failed to delete old mappings: %w", err)
	}

	// Insert new mappings using the junction type J
	var insertBuilder orm.InsertObjectQueryBuilder

	for i := 0; i < relationField.Len(); i++ {
		targetVal := relationField.Index(i)
		targetIDField := targetVal.FieldByName("ID")
		if !targetIDField.IsValid() {
			continue
		}
		tgtID := fmt.Sprintf("%v", targetIDField.Interface())

		// Create new Junction instance
		junctionPtr := reflect.New(reflect.TypeOf(*new(J)))
		junctionVal := junctionPtr.Elem()

		// Populate FKs dynamically relying on tag mappings
		// In J, we need to find which fields are mapped to JoinFK and JoinAssFK DB columns

		// We find the field indices based on `db` tag
		for i := 0; i < junctionVal.NumField(); i++ {
			fieldTag := junctionVal.Type().Field(i).Tag.Get("db")
			if fieldTag == relInfo.JoinFK {
				if junctionVal.Field(i).CanSet() {
					junctionVal.Field(i).SetString(srcID)
				}
			} else if fieldTag == relInfo.JoinAssFK {
				if junctionVal.Field(i).CanSet() {
					junctionVal.Field(i).SetString(tgtID)
				}
			}
		}

		// Build and execute the insert query utilizing junction struct
		query, args, err := insertBuilder.BuildFrom(junctionPtr.Interface())
		if err != nil {
			return fmt.Errorf("SaveManyToMany: failed to build insert query for junction: %w", err)
		}

		_, err = manager.Insert(query, args...)
		if err != nil {
			return fmt.Errorf("SaveManyToMany: failed to execute junction insert: %w", err)
		}
	}

	return nil
}
