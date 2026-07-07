package domain

import (
	"time"

	"github.com/google/uuid"
)

// Organization represents the top-level tenant entity.
type Organization struct {
	ID        uuid.UUID
	Name      string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}
