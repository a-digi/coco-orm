package orm

// RelationHandler defines generic operations for managing entity relationships (e.g., Many-to-Many).
// S represents the source entity type, and T represents the target entity type.
type RelationHandler[S any, T any] interface {
	Load(manager *DatabaseManager, sources *[]S) error
	Save(manager *DatabaseManager, source *S) error
	Delete(manager *DatabaseManager, source *S) error
}
