# SQLite3 Connector für Go

## Funktionen
- Öffnen und Schließen einer SQLite3-Datenbank
- Fehlerbehandlung und Ping-Test
- Nutzt Go-Standardbibliothek `database/sql` und den Treiber `github.com/mattn/go-sqlite3`

## Beispiel
```go
import "github.com/a-digi/coco-orm"

func main() {
    connector, err := db.NewConnector("test.db")
    if err != nil {
        panic(err)
    }
    defer connector.Close()
    // connector.DB ist eine *sql.DB Instanz
}
```

## Dynamic Queries and Pagination

The ORM offers advanced retrieval using reflection-based generic slicing and dynamic query building via the `model.SelectQuery` structure.

### `FindAllDynamic`
A non-generic version of `FindAll` allowing dynamically created `interface{}` slices, heavily useful when your entity type isn't known at compile-time:

```go
// create a dynamic slice pointer ptr to hold the objects
sliceType := reflect.SliceOf(entityType)
resultsPtr := reflect.New(sliceType)

err := db.FindAllDynamic(manager, resultsPtr.Interface())
```

### `FindByQuery`
Uses a `model.SelectQuery` wrapper to automatically handle filtering, sorting, and pagination.

```go
import "github.com/a-digi/coco-orm/orm/model"

query := model.SelectQuery{
    Entity: resultsPtr.Interface(), // Obligatory pointer to struct slice
    Pagination: model.Pagination{
        Page:  1,
        Limit: 10,
    },
    Filters: []model.Filter{
        {
            Column: "title",
            Value:  "Admin",
            Type:   model.FilterTypePartialMatch, // "LIKE %Admin%"
        },
    },
    Sorting: []model.Sorting{
        {
            Column:    "created_at",
            Direction: model.SortDirectionDesc,
        },
    },
}

err := db.FindByQuery(manager, query)
```
