package domain

import (
	"github.com/google/uuid"
	"time"
)

type Document struct {
	LinkID uuid.UUID

	URL string

	Title   string
	Content string

	IndexedAt time.Time
	PageRank  float64
}
