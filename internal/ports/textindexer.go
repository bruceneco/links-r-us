package ports

import (
	"errors"
	"github.com/bruceneco/links-r-us/internal/application/core/domain"
	"github.com/google/uuid"
)

type TextIndexer interface {
	Index(doc *domain.Document) error
	FindByID(linkID uuid.UUID) (*domain.Document, error)
	Search(query *DocumentQuery) (DocumentIterator, error)
	UpdateScore(linkID uuid.UUID, score float64) error
}

type DocumentQueryType uint8
type DocumentQuery struct {
	Type       DocumentQueryType
	Expression string
	Offset     uint64
}

const (
	DocumentQueryTypeMatch DocumentQueryType = iota
	DocumentQueryTypePhrase
)

type DocumentIterator interface {
	Close() error
	Next() bool
	Error() error
	Document() *domain.Document
	TotalCount() uint64
}

var (
	// ErrNotFound is returned by the indexer when attempting to look up
	// a document that does not exist.
	TextIndexerErrNotFound = errors.New("not found")

	// ErrMissingLinkID is returned when attempting to index a document
	// that does not specify a valid link ID.
	TextIndexerErrMissingLinkID = errors.New("document does not provide a valid linkID")
)
