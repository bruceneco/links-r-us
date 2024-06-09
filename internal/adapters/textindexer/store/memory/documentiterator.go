package memory

import (
	"github.com/blevesearch/bleve/v2"
	"github.com/bruceneco/links-r-us/internal/application/core/domain"
)

// documentIterator implements index.Iterator.
type documentIterator struct {
	idx       *InMemoryIndexer
	searchReq *bleve.SearchRequest

	cumIdx uint64
	rsIdx  int
	rs     *bleve.SearchResult

	latchedDoc *domain.Document
	lastErr    error
}

// Close the iterator and release any allocated resources.
func (it *documentIterator) Close() error {
	it.idx = nil
	it.searchReq = nil
	if it.rs != nil {
		it.cumIdx = it.rs.Total
	}
	return nil
}

// Next loads the next document matching the search query.
// It returns false if no more documents are available.
func (it *documentIterator) Next() bool {
	if it.lastErr != nil || it.rs == nil || it.cumIdx >= it.rs.Total {
		return false
	}

	// Do we need to fetch the next batch?
	if it.rsIdx >= it.rs.Hits.Len() {
		it.searchReq.From += it.searchReq.Size
		if it.rs, it.lastErr = it.idx.idx.Search(it.searchReq); it.lastErr != nil {
			return false
		}

		it.rsIdx = 0
	}

	nextID := it.rs.Hits[it.rsIdx].ID
	if it.latchedDoc, it.lastErr = it.idx.findByID(nextID); it.lastErr != nil {
		return false
	}

	it.cumIdx++
	it.rsIdx++
	return true
}

// Error returns the last error encountered by the iterator.
func (it *documentIterator) Error() error {
	return it.lastErr
}

// Document returns the current document from the result set.
func (it *documentIterator) Document() *domain.Document {
	return it.latchedDoc
}

// TotalCount returns the approximate number of search results.
func (it *documentIterator) TotalCount() uint64 {
	if it.rs == nil {
		return 0
	}
	return it.rs.Total
}
