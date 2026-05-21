package orm

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/a-digi/coco-orm/orm/metadata"
)

// AutoEagerLoadRelations dynamically inspects a slice of entities and eager loads any m2o relations.
// This is slower than strictly typed EagerLoadManyToOne because it relies entirely on reflection mapping,
// but it is extremely useful for generic REST handlers where compile-time types T and S are unknown.
func (manager *DatabaseManager) AutoEagerLoadRelations(entitiesPtr interface{}) error {
	if manager == nil || manager.Connector == nil || manager.Connector.DB == nil {
		return fmt.Errorf("DatabaseManager or Connector is nil")
	}

	if entitiesPtr == nil {
		return nil
	}

	sliceVal := reflect.ValueOf(entitiesPtr)
	if sliceVal.Kind() != reflect.Ptr || sliceVal.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("AutoEagerLoadRelations expects a pointer to a slice")
	}

	sliceElem := sliceVal.Elem()
	if sliceElem.Len() == 0 {
		return nil
	}

	// 1. Get the underlying struct type
	elemType := sliceElem.Index(0).Type()
	if elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}

	// 2. Find any m2o relations from the metadata (using a dummy instance)
	dummyObj := reflect.New(elemType).Elem().Interface()
	sourceMeta := metadata.GetDataTypeInfo(dummyObj)

	var relationsToLoad []metadata.RelationInfo
	for _, rel := range sourceMeta.Relations {
		if rel.Type == "m2o" {
			relationsToLoad = append(relationsToLoad, rel)
		}
	}

	if len(relationsToLoad) == 0 {
		return nil
	}

	// 3. Process each m2o relation
	for _, relInfo := range relationsToLoad {
		err := manager.loadDynamicM2O(sliceElem, elemType, relInfo)
		if err != nil {
			return fmt.Errorf("failed to auto load relation %s: %w", relInfo.FieldName, err)
		}
	}

	return nil
}

func (manager *DatabaseManager) loadDynamicM2O(sliceElem reflect.Value, elemType reflect.Type, relInfo metadata.RelationInfo) error {
	// Find the source field name corresponding to JoinFK
	var sourceFKFieldName string
	for i := 0; i < elemType.NumField(); i++ {
		if elemType.Field(i).Tag.Get("db") == relInfo.JoinFK {
			sourceFKFieldName = elemType.Field(i).Name
			break
		}
	}

	if sourceFKFieldName == "" {
		return fmt.Errorf("missing source FK field for db mapping %s", relInfo.JoinFK)
	}

	// Extract unique FK IDs
	fkValues := make([]interface{}, 0, sliceElem.Len())
	uniqueFKs := make(map[string]bool)

	for i := 0; i < sliceElem.Len(); i++ {
		val := sliceElem.Index(i)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
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
		return nil
	}

	placeholders := make([]string, len(fkValues))
	for i := range placeholders {
		placeholders[i] = "?"
	}

	targetTableName := relInfo.JoinTable
	if targetTableName == "" {
		return fmt.Errorf("target table not defined for m2o relation %s", relInfo.FieldName)
	}

	targetQuery := fmt.Sprintf("SELECT * FROM %s WHERE %s IN (%s)", targetTableName, relInfo.JoinAssFK, strings.Join(placeholders, ","))

	tgtRows, err := manager.Connector.DB.Query(targetQuery, fkValues...)
	if err != nil {
		return fmt.Errorf("dynamic loader failed target DB query: %w", err)
	}
	defer tgtRows.Close()

	// Determine the target struct type to dynamically slice up
	targetType := relInfo.FieldType
	if targetType.Kind() == reflect.Ptr {
		targetType = targetType.Elem()
	}

	// We need a slice of the target type to pass to HydrateRows
	targetSlicePtr := reflect.New(reflect.SliceOf(targetType))

	if err := manager.Hydrator.HydrateRows(tgtRows, targetSlicePtr.Interface()); err != nil {
		return fmt.Errorf("dynamic loader failed to hydrate target entities: %w", err)
	}

	// Map the fetched targets
	targetSlice := targetSlicePtr.Elem()

	var targetRefFieldName string
	for i := 0; i < targetType.NumField(); i++ {
		if targetType.Field(i).Tag.Get("db") == relInfo.JoinAssFK {
			targetRefFieldName = targetType.Field(i).Name
			break
		}
	}

	if targetRefFieldName == "" {
		return fmt.Errorf("missing target ref field for db mapping %s", relInfo.JoinAssFK)
	}

	targetMap := make(map[string]reflect.Value)
	for i := 0; i < targetSlice.Len(); i++ {
		tEntity := targetSlice.Index(i)
		refField := tEntity.FieldByName(targetRefFieldName)
		if refField.IsValid() {
			refID := fmt.Sprintf("%v", refField.Interface())
			// We store pointers to the target elements
			targetMap[refID] = tEntity.Addr()
		}
	}

	// Distribute pointers back to source instances
	for i := 0; i < sliceElem.Len(); i++ {
		elemVal := sliceElem.Index(i)
		if elemVal.Kind() == reflect.Ptr {
			elemVal = elemVal.Elem()
		}

		fkField := elemVal.FieldByName(sourceFKFieldName)
		if !fkField.IsValid() || fkField.IsZero() {
			continue
		}

		fkStr := fmt.Sprintf("%v", fkField.Interface())

		if tPtr, exists := targetMap[fkStr]; exists {
			fieldVal := elemVal.FieldByName(relInfo.FieldName)
			if fieldVal.CanSet() {
				fieldVal.Set(tPtr)
			}
		}
	}

	return nil
}

// AutoEagerLoadM2MRelations dynamically inspects a slice of entities and eager loads any m2m relations.
func (manager *DatabaseManager) AutoEagerLoadM2MRelations(entitiesPtr interface{}) error {
	if manager == nil || manager.Connector == nil || manager.Connector.DB == nil {
		return fmt.Errorf("DatabaseManager or Connector is nil")
	}

	if entitiesPtr == nil {
		return nil
	}

	sliceVal := reflect.ValueOf(entitiesPtr)
	if sliceVal.Kind() != reflect.Ptr || sliceVal.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("AutoEagerLoadM2MRelations expects a pointer to a slice")
	}

	sliceElem := sliceVal.Elem()
	if sliceElem.Len() == 0 {
		return nil
	}

	elemType := sliceElem.Index(0).Type()
	if elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}

	dummyObj := reflect.New(elemType).Elem().Interface()
	sourceMeta := metadata.GetDataTypeInfo(dummyObj)

	var m2mRelations []metadata.RelationInfo
	for _, rel := range sourceMeta.Relations {
		if rel.Type == "m2m" {
			m2mRelations = append(m2mRelations, rel)
		}
	}

	if len(m2mRelations) == 0 {
		return nil
	}

	for _, relInfo := range m2mRelations {
		err := manager.loadDynamicM2M(sliceElem, elemType, relInfo)
		if err != nil {
			return fmt.Errorf("failed to auto load m2m relation %s: %w", relInfo.FieldName, err)
		}
	}

	return nil
}

func (manager *DatabaseManager) loadDynamicM2M(sliceElem reflect.Value, elemType reflect.Type, relInfo metadata.RelationInfo) error {
	// 1. Extract IDs from sources
	sourceIDs := make([]interface{}, 0, sliceElem.Len())
	for i := 0; i < sliceElem.Len(); i++ {
		val := sliceElem.Index(i)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
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

	// 2. Pivot lookup
	queryPivot := fmt.Sprintf("SELECT %s, %s FROM %s WHERE %s IN (%s)",
		relInfo.JoinFK, relInfo.JoinAssFK, relInfo.JoinTable, relInfo.JoinFK, strings.Join(placeholders, ","))

	rows, err := manager.Connector.DB.Query(queryPivot, sourceIDs...)
	if err != nil {
		return fmt.Errorf("loadDynamicM2M pivot error: %w", err)
	}
	defer rows.Close()

	sourceToTargetIDs := make(map[string][]string)
	uniqueTargetIDs := make(map[string]bool)

	for rows.Next() {
		var srcID, tgtID string
		if err := rows.Scan(&srcID, &tgtID); err != nil {
			return err
		}
		sourceToTargetIDs[srcID] = append(sourceToTargetIDs[srcID], tgtID)
		uniqueTargetIDs[tgtID] = true
	}

	if len(uniqueTargetIDs) == 0 {
		return nil
	}

	// 3. Targets lookup
	targetType := relInfo.FieldType.Elem()
	if targetType.Kind() == reflect.Ptr {
		targetType = targetType.Elem()
	}

	dummyT := reflect.New(targetType).Elem().Interface()
	targetTableName, ok := metadata.GetTableName(dummyT)
	if !ok || targetTableName == "" {
		return fmt.Errorf("target table not found")
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
		return fmt.Errorf("loadDynamicM2M target query err: %w", err)
	}
	defer tgtRows.Close()

	// 4. Hydration
	targetSlicePtr := reflect.New(reflect.SliceOf(targetType))
	if err := manager.Hydrator.HydrateRows(tgtRows, targetSlicePtr.Interface()); err != nil {
		return err
	}

	targetSlice := targetSlicePtr.Elem()
	targetMap := make(map[string]reflect.Value)
	for i := 0; i < targetSlice.Len(); i++ {
		tEntity := targetSlice.Index(i)
		idField := tEntity.FieldByName("ID")
		if idField.IsValid() {
			tgtID := fmt.Sprintf("%v", idField.Interface())
			targetMap[tgtID] = tEntity
		}
	}

	// 5. Assignment
	for i := 0; i < sliceElem.Len(); i++ {
		elemVal := sliceElem.Index(i)
		if elemVal.Kind() == reflect.Ptr {
			elemVal = elemVal.Elem()
		}

		srcIDField := elemVal.FieldByName("ID")
		if !srcIDField.IsValid() {
			continue
		}
		srcID := fmt.Sprintf("%v", srcIDField.Interface())

		associatedTgtIDs, ok := sourceToTargetIDs[srcID]
		if !ok || len(associatedTgtIDs) == 0 {
			continue
		}

		associatedSlice := reflect.MakeSlice(relInfo.FieldType, 0, len(associatedTgtIDs))
		for _, tgtID := range associatedTgtIDs {
			if tEntity, exists := targetMap[tgtID]; exists {
				if relInfo.FieldType.Elem().Kind() == reflect.Ptr {
					associatedSlice = reflect.Append(associatedSlice, tEntity.Addr())
				} else {
					associatedSlice = reflect.Append(associatedSlice, tEntity)
				}
			}
		}

		if associatedSlice.Len() > 0 {
			fieldVal := elemVal.FieldByName(relInfo.FieldName)
			if fieldVal.CanSet() {
				fieldVal.Set(associatedSlice)
			}
		}
	}

	return nil
}
