# Many-to-Many Annotation Implementation Plan (Generics & Custom Junction Entities)

This document outlines a step-by-step plan to introduce many-to-many (`m2m`) relation annotations in `coco-orm`, emphasizing a **true generic implementation** leveraging Go's `[T any]` type parameters and supporting explicit standard entities for **Junction/Pivot Tables**.

## 1. Goal & Tag Design

To properly manage a many-to-many relationship, the ORM needs metadata mapping entities through a junction (or pivot) table. Often, this junction table is more than just a blind mapping and acts as an entity itself (e.g. `UserGroupMember`) with UUIDs, active status, and timestamps.

### Example Struct Usage

```go
// Primary Entity (e.g., User)
type User struct {
	_            struct{} `table:"admin_users"`
	ID           string   `db:"id" dbtype:"UUID" nullable:"true" json:"id"`
	// ... other fields
	
	// Many-to-many relation defining the joining logic and the entity used for the junction table
	Groups []Group `relation:"m2m" join_entity:"UserGroupMember" join_table:"user_group_members" join_fk:"user_id" join_ass_fk:"group_id" json:"groups,omitempty"`
}

// Junction Entity (e.g., UserGroupMember)
type UserGroupMember struct {
	_         struct{} `table:"user_group_members"`
	ID        string   `db:"id" dbtype:"UUID" nullable:"false" json:"id"`
	GroupID   string   `db:"group_id" dbtype:"TEXT" nullable:"false" json:"group_id"`
	UserID    string   `db:"user_id" dbtype:"TEXT" nullable:"false" json:"user_id"`
	CreatedAt string   `db:"created_at" dbtype:"DATETIME" nullable:"true" json:"created_at"`
	IsActive  bool     `db:"is_active" dbtype:"BOOLEAN" nullable:"false" default:"true" json:"is_active"`
}
```

- `relation`: Defines the type of relation (`m2m`).
- `join_entity`: (Optional) The name/type of the struct representing the junction record so the ORM can hydrate/insert it properly (triggering `ID` creation, picking default fields).
- `join_table`: The database table for the junction.
- `join_fk`: The column referencing the source entity (`User.ID`).
- `join_ass_fk`: The column referencing the target entity (`Group.ID`).

## 2. Metadata Extraction Update (`orm/metadata/`)

**Steps:**
1. In `metadata/datatype.go`, create a `RelationInfo` struct:
   ```go
   type RelationInfo struct {
       Type          string // "m2m"
       FieldType     reflect.Type
       FieldName     string 
       JoinEntity    string // "UserGroupMember"
       JoinTable     string // "user_group_members"
       JoinFK        string // "user_id"
       JoinAssFK     string // "group_id"
   }
   ```
2. Update `DataTypeInfo` to include `Relations []RelationInfo`.
3. In `GetDataTypeInfo()`, add parsing for the `relation` tag. Exclude relational slice fields from regular database `Columns`.

## 3. Abstracting the Relation Handling (Interfaces & Generics)

Create an interface `RelationHandler[S, T any]` inside the `orm` package.
```go
type RelationHandler[S any, T any] interface {
    Load(manager *DatabaseManager, sources []S) error
    Save(manager *DatabaseManager, source *S) error
    Delete(manager *DatabaseManager, source *S) error
}
```
A specific `ManyToManyHandler[S, T, J any]` (where `J` is the Junction Entity type, or a runtime mapped type if strictly generic across the board) would implement this, utilizing the metadata.

## 4. Generic Query Builder and Eager Loading (`orm/dbmanager.go`)

**Steps:**
1. Create a generic function to eagerly load a many-to-many relation.
   ```go
   func EagerLoadManyToMany[S any, T any](manager *DatabaseManager, sources *[]S, relationName string) error {
       // Extract S IDs
       // SELECT target.* FROM groups target INNER JOIN user_group_members pivot ON pivot.group_id = target.id WHERE pivot.user_id IN (?)
       // Hydrate rows to map T by S.ID and assign back to S.Groups.
   }
   ```
2. For explicit hydration filtering where intermediate fields matter, provide `EagerLoadManyToManyWithJunction[S, T, J any]`.

## 5. Insert, Update, and Delete Routines (`orm/dbmanager.go`)

Relations require atomic junction data persistence logic, utilizing generics for type safety on save operations and triggering table-specific lifecycle logic for junction entities (e.g. UUID generation for `UserGroupMember.ID`).

**Steps:**
1. **Generic Save:** When saving `User`, intercept the `Groups` slice handling.
2. **Insert Junction:** 
   - Insert `User` first to get its primary key.
   - For every `Group`, create an instance of `Junction Type` (e.g. `UserGroupMember`).
   - Populate `UserID` and `GroupID`, and ask the `manager` to explicitly dynamically save the junction instance. This ensures UUIDs are generated and defaults (like `CreatedAt`) form accurately.
3. **Update:** 
   - Clear stale links in `join_table` where `join_fk = sourceID`.
   - Iterate through updated `*T` slice's relations and re-insert the active links as discrete instances of `Junction Type`.
4. **Delete:** 
   - Delete `join_table` rows mapped to the `User`.

## 6. Modifying Generic Hydration (`orm/hydration.go`)

Update the hydrator to work better with Go generics.
```go
func HydrateGeneric[T any](rows *sql.Rows, dest *[]T) error
```

## 7. Writing Tests (`tests/`)

**Steps:**
1. Create a suite file `tests/m2m_generic_test.go`.
2. Emulate an `User` struct matching `Group` through the explicit junction entity `UserGroupMember` utilizing db methods `FindAll[User]`.
3. Persist a mock `User` with two `Group` objects. Verify the intermediate rows inserted into `user_group_members` possess valid UUIDs for their own `ID` columns instead of failing constraints.
4. Load the stored `User` using `FindByQuery[User]`, specifying the relation field to eager load, ensuring `[]Group` mapping accurately populates.
