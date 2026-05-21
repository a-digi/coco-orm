package orm

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/a-digi/coco-orm/orm/metadata"
)

// EagerLoadManyToOne loads many-to-one relations for a slice of source entities.
// S is the source entity type, T is the target entity type being pointed to.
func EagerLoadManyToOne[S any, T any](manager *DatabaseManager, sources *[]S, relationFieldName string) error {
	if manager == nil || manager.Connector == nil || manager.Connector.DB == nil {
		return fmt.Errorf("DatabaseManager or Connector is nil")
	}

	if sources == nil || len(*sources) == 0 {
		return nil
	}

	// 1. Get Source Metadata to find the relation
	var dummyS S
	sourceMeta := metadata.GetDataTypeInfo(dummyS)
	var relInfo *metadata.RelationInfo

	for _, rel := range sourceMeta.Relations {
		if rel.FieldName == relationFieldName {
			relInfo = &rel
			break
		}
	}

	if relInfo == nil {
		return fmt.Errorf("relation %s not found in source entity", relationFieldName)
	}

	if relInfo.Type != "m2o" {
		return fmt.Errorf("relation %s is not m2o", relationFieldName)
	}

	// 2. Identify the field on the Source holding the Foreign Key (relInfo.JoinFK maps to a DB column)
	var sourceFKFieldName string
	// We need to look at dummyS fields and find the one where the tag `db` equals relInfo.JoinFK
	sValType := reflect.TypeOf(dummyS)
	if sValType.Kind() == reflect.Ptr {
		sValType = sValType.Elem()
	}
	for i := 0; i < sValType.NumField(); i++ {
		dbTag := sValType.Field(i).Tag.Get("db")
		if dbTag == relInfo.JoinFK {
			sourceFKFieldName = sValType.Field(i).Name
			break
		}
	}

	if sourceFKFieldName == "" {
		return fmt.Errorf("could not find source struct field corresponding to join_fk DB column %s", relInfo.JoinFK)
	}

	// 3. Extract the Foreign Key values from all source entities
	fkValues := make([]interface{}, 0, len(*sources))
	uniqueFKs := make(map[string]bool)

	for i := 0; i < len(*sources); i++ {
		val := reflect.ValueOf((*sources)[i])
		fkField := val.FieldByName(sourceFKFieldName)
		if fkField.IsValid() && !fkField.IsZero() {
			fkStr := fmt.Sprintf("%v", fkField.Interface())
			if !uniqueFKs[fkStr] {
				uniqueFKs[fkStr] = true
				fkValues = append(fkValues, fkField.Interface())
			}
		}
	}

	if len(fkValues) == 0 {
		return nil // No foreign keys to lookup
	}

	placeholders := make([]string, len(fkValues))
	for i := range placeholders {
		placeholders[i] = "?"
	}

	// 4. Query the target entities using the join_ass_fk (e.g. `id`) column
	var dummyT T
	targetTableName, ok := metadata.GetTableName(dummyT)
	if !ok || targetTableName == "" {
		// Fallback to the join_table from the tag
		targetTableName = relInfo.JoinTable
		if targetTableName == "" {
			return fmt.Errorf("target table not found for target entity")
		}
	}

	targetQuery := fmt.Sprintf("SELECT * FROM %s WHERE %s IN (%s)", targetTableName, relInfo.JoinAssFK, strings.Join(placeholders, ","))

	tgtRows, err := manager.Connector.DB.Query(targetQuery, fkValues...)
	if err != nil {
		return fmt.Errorf("EagerLoadManyToOne: failed to query target entities: %w", err)
	}
	defer tgtRows.Close()

	var targetEntities []T
	if err := manager.Hydrator.HydrateRows(tgtRows, &targetEntities); err != nil {
		return fmt.Errorf("EagerLoadManyToOne: failed to hydrate target entities: %w", err)
	}

	// 5. Map target entities by their reference ID (join_ass_fk translated back to the T struct field)
	var targetRefFieldName string
	tValType := reflect.TypeOf(dummyT)
	if tValType.Kind() == reflect.Ptr {
		tValType = tValType.Elem()
	}
	for i := 0; i < tValType.NumField(); i++ {
		dbTag := tValType.Field(i).Tag.Get("db")
		if dbTag == relInfo.JoinAssFK {
			targetRefFieldName = tValType.Field(i).Name
			break
		}
	}

	if targetRefFieldName == "" {
		return fmt.Errorf("could not find target struct field corresponding to join_ass_fk DB column %s", relInfo.JoinAssFK)
	}

	targetMap := make(map[string]*T)
	for i := range targetEntities {
		tEntity := &targetEntities[i]
		val := reflect.ValueOf(*tEntity)
		refField := val.FieldByName(targetRefFieldName)
		if refField.IsValid() {
			refID := fmt.Sprintf("%v", refField.Interface())
			targetMap[refID] = tEntity
		}
	}

	// 6. Assign Target pointers back to Source structs
	sliceVal := reflect.ValueOf(sources).Elem()
	for i := 0; i < sliceVal.Len(); i++ {
		elemVal := sliceVal.Index(i)

		fkField := elemVal.FieldByName(sourceFKFieldName)
		if !fkField.IsValid() || fkField.IsZero() {
			continue
		}

		fkStr := fmt.Sprintf("%v", fkField.Interface())

		if tPtr, exists := targetMap[fkStr]; exists {
			fieldVal := elemVal.FieldByName(relationFieldName)
			if fieldVal.CanSet() {
				fieldVal.Set(reflect.ValueOf(tPtr))
			}
		}
	}

	return nil
}
