package domain

import (
	"github.com/google/uuid"
	"time"
)

// Link is a representation of a URL in WWW
type Link struct {
	// ID is the UUID that uniquely identifies the Link.
	ID uuid.UUID
	// URL of the Link.
	URL string
	// RetrievedAt is when the Link was retrieved.
	RetrievedAt time.Time
}
