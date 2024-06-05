package ports

import "github.com/bruceneco/links-r-us/internal/application/core/domain"

type EdgeRepository interface {
	Upsert(edge *domain.Edge) error
}
