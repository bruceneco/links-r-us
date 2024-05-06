package ports

import (
	"github.com/bruceneco/links-r-us/internal/application/core/domain"
	"github.com/google/uuid"
	"time"
)

// Graph is the port to manage the relation between many domain.Link and their domain.Edge.
type Graph interface {
	// UpsertLink updates an existing entry or insert it if it does not exist.
	UpsertLink(link *domain.Link) error
	// FindLink retrieves a domain.Link using its uuid.
	FindLink(id uuid.UUID) (*domain.Link, error)

	// UpsertEdge updates an existing Edge or insert it if it does not exist.
	UpsertEdge(edge *domain.Edge) error
	// RemoveStaleEdges deletes domain.Edge based on its uuid and the last update.
	RemoveStaleEdges(fromID uuid.UUID, updatedBefore time.Time) error

	// Links retrieves a list of domain.Link from an uuid to another uuid based on their retrieved datetime.
	Links(fromID, toID uuid.UUID, retrievedBefore time.Time) (LinkIterator, error)
	// Edges retrieves a list of domain.Edge from an uuid to another uuid based on their update datetime.
	Edges(fromID, toID uuid.UUID, updatedBefore time.Time) (EdgeIterator, error)
}

// LinkIterator is implemented by structs that can iterate a list of domain.Link from graph.
type LinkIterator interface {
	Iterator

	// Link returns the currently fetched domain.Link struct.
	Link() *domain.Link
}

// EdgeIterator is implemented by structs that can iterate a list of domain.Edge from graph.
type EdgeIterator interface {
	Iterator
	// Edge returns the currently fetched domain.Edge struct.
	Edge() *domain.Edge
}
