package entity

import (
	"time"

	"github.com/uptrace/bun"
)

// Order represents a purchase order stored in the relational database.
type Order struct {
	bun.BaseModel `bun:"table:orders"`

	ID        int64     `bun:",pk,autoincrement"`
	Number    string    `bun:"number"`
	Status    string    `bun:"status"`
	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `bun:"updated_at,nullzero"`
}
