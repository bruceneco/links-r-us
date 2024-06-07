package ports

import (
	"github.com/bruceneco/links-r-us/internal/application/core/domain"
	"github.com/bruceneco/links-r-us/internal/ports"
	"github.com/google/uuid"
	"time"
)

type EdgeRepository interface {
	Upsert(edge *domain.Edge) error
	Edges(fromID, toID uuid.UUID, updatedBefore time.Time) (ports.EdgeIterator, error)
}
