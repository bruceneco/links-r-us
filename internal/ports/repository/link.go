package ports

import (
	"github.com/bruceneco/links-r-us/internal/application/core/domain"
	"github.com/google/uuid"
)

type LinkRepository interface {
	Upsert(link *domain.Link) error
	Find(id uuid.UUID) (*domain.Link, error)
}
