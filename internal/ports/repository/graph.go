package ports

import (
	"github.com/bruceneco/links-r-us/internal/application/core/domain"
	"github.com/bruceneco/links-r-us/internal/ports"
	"github.com/google/uuid"
	"time"
)

type LinkRepository interface {
	Upsert(link *domain.Link) error
	Find(id uuid.UUID) (*domain.Link, error)
	Links(fromId, toId uuid.UUID, accessedBefore time.Time) (ports.LinkIterator, error)
}

type EdgeRepository interface {
	Upsert(edge *domain.Edge) error
	Edges(fromID, toID uuid.UUID, updatedBefore time.Time) (ports.EdgeIterator, error)
	RemoveStaleEdges(fromID uuid.UUID, updatedBefore time.Time) error
}