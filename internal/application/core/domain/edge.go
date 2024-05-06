package domain

import (
	"github.com/google/uuid"
	"time"
)

// Edge is the representation of the connection between two Link.
type Edge struct {
	// ID is the UUID that uniquely identifies the Edge.
	ID uuid.UUID
	// Src is the UUID belonged to the Link from where the Edge was discovered.
	Src uuid.UUID
	// Dst is the UUID belonged to the Link to where the Edge will point at.
	Dst uuid.UUID
	// UpdatedAt is the datetime of last Edge visiting.
	UpdatedAt time.Time
}
