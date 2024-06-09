package memory

import (
	"fmt"
	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/bruceneco/links-r-us/internal/application/core/domain"
	"github.com/bruceneco/links-r-us/internal/ports"
	"github.com/google/uuid"
	"sync"
	"time"
)

// The size of each page of results that is cached locally by the iterator.
const batchSize = 10

type bleveDoc struct {
	Title    string
	Content  string
	PageRank float64
}

// InMemoryIndexer is an Indexer implementation that uses an in-memory
// bleve instance to catalogue and search documents.
type InMemoryIndexer struct {
	mu   sync.RWMutex
	docs map[string]*domain.Document

	idx bleve.Index
}

// NewInMemoryIndexer creates a text indexer that uses an in-memory
// bleve instance for indexing documents.
func NewInMemoryIndexer() (ports.TextIndexer, error) {
	mapping := bleve.NewIndexMapping()
	idx, err := bleve.NewMemOnly(mapping)
	if err != nil {
		return nil, err
	}

	return &InMemoryIndexer{
		idx:  idx,
		docs: make(map[string]*domain.Document),
	}, nil
}

// Close the indexer and release any allocated resources.
func (i *InMemoryIndexer) Close() error {
	return i.idx.Close()
}

// Index inserts a new document to the index or updates the index entry
// for and existing document.
func (i *InMemoryIndexer) Index(doc *domain.Document) error {
	if doc.LinkID == uuid.Nil {
		return fmt.Errorf("index: %w", ports.TextIndexerErrMissingLinkID)
	}

	doc.IndexedAt = time.Now()
	dcopy := copyDoc(doc)
	key := dcopy.LinkID.String()

	i.mu.Lock()
	defer i.mu.Unlock()

	// If updating, preserve existing PageRank score
	if orig, exists := i.docs[key]; exists {
		dcopy.PageRank = orig.PageRank
	}

	if err := i.idx.Index(key, makeBleveDoc(dcopy)); err != nil {
		return fmt.Errorf("index: %w", err)
	}

	i.docs[key] = dcopy
	return nil
}

// FindByID looks up a document by its link ID.
func (i *InMemoryIndexer) FindByID(linkID uuid.UUID) (*domain.Document, error) {
	return i.findByID(linkID.String())
}

// findByID looks up a document by its link UUID expressed as a string.
func (i *InMemoryIndexer) findByID(linkID string) (*domain.Document, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	if d, found := i.docs[linkID]; found {
		return copyDoc(d), nil
	}

	return nil, fmt.Errorf("find by ID: %w", ports.TextIndexerErrNotFound)
}

// Search the index for a particular query and return back a result
// iterator.
func (i *InMemoryIndexer) Search(q *ports.DocumentQuery) (ports.DocumentIterator, error) {
	var bq query.Query
	switch q.Type {
	case ports.DocumentQueryTypePhrase:
		bq = bleve.NewMatchPhraseQuery(q.Expression)
	default:
		bq = bleve.NewMatchQuery(q.Expression)
	}

	searchReq := bleve.NewSearchRequest(bq)
	searchReq.SortBy([]string{"-PageRank", "-_score"})
	searchReq.Size = batchSize
	searchReq.From = int(q.Offset)
	rs, err := i.idx.Search(searchReq)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	return &documentIterator{idx: i, searchReq: searchReq, rs: rs, cumIdx: q.Offset}, nil
}

// UpdateScore updates the PageRank score for a document with the specified
// link ID. If no such document exists, a placeholder document with the
// provided score will be created.
func (i *InMemoryIndexer) UpdateScore(linkID uuid.UUID, score float64) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	key := linkID.String()
	doc, found := i.docs[key]
	if !found {
		doc = &domain.Document{LinkID: linkID}
		i.docs[key] = doc
	}

	doc.PageRank = score
	if err := i.idx.Index(key, makeBleveDoc(doc)); err != nil {
		return fmt.Errorf("update score: %w", err)
	}

	return nil
}

func copyDoc(d *domain.Document) *domain.Document {
	dcopy := new(domain.Document)
	*dcopy = *d
	return dcopy
}

func makeBleveDoc(d *domain.Document) bleveDoc {
	return bleveDoc{
		Title:    d.Title,
		Content:  d.Content,
		PageRank: d.PageRank,
	}
}
