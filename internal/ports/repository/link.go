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
