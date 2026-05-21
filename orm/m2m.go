package orm

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/a-digi/coco-orm/orm/metadata"
)

// EagerLoadManyToMany loads many-to-many relations for a slice of source entities.
// S is the source entity type, T is the target entity type.
func EagerLoadManyToMany[S any, T any](manager *DatabaseManager, sources *[]S, relationFieldName string) error {
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

	if relInfo.Type != "m2m" {
		return fmt.Errorf("relation %s is not m2m", relationFieldName)
	}

	// 2. Extract IDs from sources
	sourceIDs := make([]interface{}, 0, len(*sources))
	for i := 0; i < len(*sources); i++ {
		val := reflect.ValueOf((*sources)[i])
		idField := val.FieldByName("ID")
		if idField.IsValid() {
			sourceIDs = append(sourceIDs, idField.Interface())
		}
	}

	if len(sourceIDs) == 0 {
		return nil
	}

	placeholders := make([]string, len(sourceIDs))
	for i := range placeholders {
		placeholders[i] = "?"
	}

	// 3. Query the junction table to get the mapping
	queryPivot := fmt.Sprintf("SELECT %s, %s FROM %s WHERE %s IN (%s)",
		relInfo.JoinFK, relInfo.JoinAssFK, relInfo.JoinTable, relInfo.JoinFK, strings.Join(placeholders, ","))

	rows, err := manager.Connector.DB.Query(queryPivot, sourceIDs...)
	if err != nil {
		return fmt.Errorf("EagerLoadManyToMany: failed to query pivot table: %w", err)
	}
	defer rows.Close()

	// Map of sourceID -> []targetID
	sourceToTargetIDs := make(map[string][]string)
	uniqueTargetIDs := make(map[string]bool)

	for rows.Next() {
		var srcID string
		var tgtID string
		if err := rows.Scan(&srcID, &tgtID); err != nil {
			return fmt.Errorf("EagerLoadManyToMany: failed to scan pivot row: %w", err)
		}
		sourceToTargetIDs[srcID] = append(sourceToTargetIDs[srcID], tgtID)
		uniqueTargetIDs[tgtID] = true
	}

	if len(uniqueTargetIDs) == 0 {
		return nil
	}

	// 4. Query the target entities
	var dummyT T
	targetTableName, ok := metadata.GetTableName(dummyT)
	if !ok || targetTableName == "" {
		return fmt.Errorf("target table not found for target entity")
	}

	targetIDs := make([]interface{}, 0, len(uniqueTargetIDs))
	for id := range uniqueTargetIDs {
		targetIDs = append(targetIDs, id)
	}

	tgtPlaceholders := make([]string, len(targetIDs))
	for i := range tgtPlaceholders {
		tgtPlaceholders[i] = "?"
	}

	targetQuery := fmt.Sprintf("SELECT * FROM %s WHERE id IN (%s)", targetTableName, strings.Join(tgtPlaceholders, ","))

	tgtRows, err := manager.Connector.DB.Query(targetQuery, targetIDs...)
	if err != nil {
		return fmt.Errorf("EagerLoadManyToMany: failed to query target entities: %w", err)
	}
	defer tgtRows.Close()

	var targetEntities []T
	if err := manager.Hydrator.HydrateRows(tgtRows, &targetEntities); err != nil {
		return fmt.Errorf("EagerLoadManyToMany: failed to hydrate target entities: %w", err)
	}

	// 5. Map target entities by their ID
	targetMap := make(map[string]T)
	for _, tEntity := range targetEntities {
		val := reflect.ValueOf(tEntity)
		idField := val.FieldByName("ID")
		if idField.IsValid() {
			tgtID := fmt.Sprintf("%v", idField.Interface())
			targetMap[tgtID] = tEntity
		}
	}

	// 6. Assign back to sources
	sliceVal := reflect.ValueOf(sources).Elem()
	for i := 0; i < sliceVal.Len(); i++ {
		elemVal := sliceVal.Index(i)
		srcIDField := elemVal.FieldByName("ID")
		if !srcIDField.IsValid() {
			continue
		}
		srcID := fmt.Sprintf("%v", srcIDField.Interface())

		associatedTgtIDs, ok := sourceToTargetIDs[srcID]
		if !ok || len(associatedTgtIDs) == 0 {
			continue
		}

		associatedEntities := make([]T, 0, len(associatedTgtIDs))
		for _, tgtID := range associatedTgtIDs {
			if tEntity, exists := targetMap[tgtID]; exists {
				associatedEntities = append(associatedEntities, tEntity)
			}
		}

		if len(associatedEntities) > 0 {
			fieldVal := elemVal.FieldByName(relationFieldName)
			if fieldVal.CanSet() {
				fieldVal.Set(reflect.ValueOf(associatedEntities))
			}
		}
	}

	return nil
}
