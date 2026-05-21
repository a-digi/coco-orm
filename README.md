# coco-orm

coco-orm ist eine leichtgewichtige, flexible und performante ORM-Bibliothek für Go (Golang). Sie bietet eine einfache Möglichkeit, mit relationalen Datenbanken zu arbeiten, ohne auf die Vorteile von Go verzichten zu müssen.

## Features

- Intuitive API für Datenbankabfragen und -operationen
- Unterstützung für verschiedene SQL-Datenbanken
- Automatische Hydration von Structs
- Migrationstools
- Erweiterbar und modular aufgebaut

## Installation

Fügen Sie coco-orm zu Ihrem Projekt hinzu:

```
go get github.com/a-digi/coco-orm
```

## Schnellstart

```go
package main

import (
    "github.com/a-digi/coco-orm/db"
)

func main() {
    manager, err := db.NewDBManager("sqlite3", "./test.db")
    if err != nil {
        panic(err)
    }
    // ... weitere ORM-Operationen ...
}
```

## Dokumentation

Die vollständige Dokumentation finden Sie im [GitHub-Repository](https://github.com/a-digi/coco-orm).

## Beispiel

```go
type User struct {
    ID   int    `coco:"primary_key"`
    Name string
}

// Tabelle registrieren und migrieren
manager.RegisterTable(&User{})
manager.Migrate()

// Einfügen
user := User{Name: "Max"}
manager.Insert(&user)

// Abfragen
var users []User
manager.Query(&users, "SELECT * FROM users")
```

## Lizenz

MIT License. Siehe [LICENSE](LICENSE) für Details.
