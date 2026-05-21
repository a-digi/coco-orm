package orm

import "time"

type Migration struct {
	ID        string
	CreatedAt time.Time
}
