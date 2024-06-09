package indextest

import (
	"errors"
	"fmt"
	"github.com/bruceneco/links-r-us/internal/application/core/domain"
	"github.com/bruceneco/links-r-us/internal/ports"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
)

// SuiteBase defines a re-usable set of index-related tests that can
// be executed against any type that implements ports.TextIndexer.
type SuiteBase struct {
	idx ports.TextIndexer
}

// SetIndexer configures the test-suite to run all tests against idx.
func (s *SuiteBase) SetIndexer(idx ports.TextIndexer) {
	s.idx = idx
}

// TestIndexDocument verifies the indexing logic for new and existing documents.
func (s *SuiteBase) TestIndexDocument(t *testing.T) {
	// Insert new Document
	doc := &domain.Document{
		LinkID:    uuid.New(),
		URL:       "https://example.com",
		Title:     "Illustrious examples",
		Content:   "Lorem ipsum dolor",
		IndexedAt: time.Now().Add(-12 * time.Hour).UTC(),
	}

	err := s.idx.Index(doc)
	assert.Nil(t, err)

	// Update existing Document
	updatedDoc := &domain.Document{
		LinkID:    doc.LinkID,
		URL:       "https://example.com",
		Title:     "A more exciting title",
		Content:   "Ovidius poeta in terra pontica",
		IndexedAt: time.Now().UTC(),
	}

	err = s.idx.Index(updatedDoc)
	assert.Nil(t, err)

	// Insert document without an ID
	incompleteDoc := &domain.Document{
		URL: "https://example.com",
	}

	err = s.idx.Index(incompleteDoc)
	assert.True(t, errors.Is(err, ports.TextIndexerErrMissingLinkID))
}

// TestIndexDoesNotOverridePageRank verifies the indexing logic for new and
// existing documents.
func (s *SuiteBase) TestIndexDoesNotOverridePageRank(t *testing.T) {
	// Insert new Document
	doc := &domain.Document{
		LinkID:    uuid.New(),
		URL:       "https://example.com",
		Title:     "Illustrious examples",
		Content:   "Lorem ipsum dolor",
		IndexedAt: time.Now().Add(-12 * time.Hour).UTC(),
	}

	err := s.idx.Index(doc)
	assert.Nil(t, err)

	// Update its score
	expScore := 0.5
	err = s.idx.UpdateScore(doc.LinkID, expScore)
	assert.Nil(t, err)

	// Update document
	updatedDoc := &domain.Document{
		LinkID:    doc.LinkID,
		URL:       "https://example.com",
		Title:     "A more exciting title",
		Content:   "Ovidius poeta in terra pontica",
		IndexedAt: time.Now().UTC(),
	}

	err = s.idx.Index(updatedDoc)
	assert.Nil(t, err)

	// Lookup document and verify that PageRank score has not been changed.
	got, err := s.idx.FindByID(doc.LinkID)
	assert.Nil(t, err)
	assert.Equal(t, got.PageRank, expScore)
}

// TestFindByID verifies the document lookup logic.
func (s *SuiteBase) TestFindByID(t *testing.T) {
	doc := &domain.Document{
		LinkID:    uuid.New(),
		URL:       "https://example.com",
		Title:     "Illustrious examples",
		Content:   "Lorem ipsum dolor",
		IndexedAt: time.Now().Add(-12 * time.Hour).UTC(),
	}

	err := s.idx.Index(doc)
	assert.Nil(t, err)

	// Look up doc
	got, err := s.idx.FindByID(doc.LinkID)
	assert.Nil(t, err)
	assert.True(t, reflect.DeepEqual(got, doc), "document returned by FindByID does not match inserted document")

	// Look up unknown
	_, err = s.idx.FindByID(uuid.New())
	assert.True(t, errors.Is(err, ports.TextIndexerErrNotFound))
}

// TestPhraseSearch verifies the document search logic when searching for
// exact phrases.
func (s *SuiteBase) TestPhraseSearch(t *testing.T) {
	var (
		numDocs = 50
		expIDs  []uuid.UUID
	)
	for i := 0; i < numDocs; i++ {
		id := uuid.New()
		doc := &domain.Document{
			LinkID:  id,
			Title:   fmt.Sprintf("doc with ID %s", id.String()),
			Content: "Lorem Ipsum Dolor",
		}

		if i%5 == 0 {
			doc.Content = "Lorem Dolor Ipsum"
			expIDs = append(expIDs, id)
		}

		err := s.idx.Index(doc)
		assert.Nil(t, err)

		err = s.idx.UpdateScore(id, float64(numDocs-i))
		assert.Nil(t, err)
	}

	it, err := s.idx.Search(&ports.DocumentQuery{
		Type:       ports.DocumentQueryTypePhrase,
		Expression: "lorem dolor ipsum",
	})
	assert.Nil(t, err)
	assert.Equal(t, iterateDocs(t, it), expIDs)
}

// TestMatchSearch verifies the document search logic when searching for
// keyword matches.
func (s *SuiteBase) TestMatchSearch(t *testing.T) {
	var (
		numDocs = 50
		expIDs  []uuid.UUID
	)
	for i := 0; i < numDocs; i++ {
		id := uuid.New()
		doc := &domain.Document{
			LinkID:  id,
			Title:   fmt.Sprintf("doc with ID %s", id.String()),
			Content: "Ovidius poeta in terra pontica",
		}

		if i%5 == 0 {
			doc.Content = "Lorem Dolor Ipsum"
			expIDs = append(expIDs, id)
		}

		err := s.idx.Index(doc)
		assert.Nil(t, err)

		err = s.idx.UpdateScore(id, float64(numDocs-i))
		assert.Nil(t, err)
	}

	it, err := s.idx.Search(&ports.DocumentQuery{
		Type:       ports.DocumentQueryTypeMatch,
		Expression: "lorem ipsum",
	})
	assert.Nil(t, err)
	assert.True(t, reflect.DeepEqual(iterateDocs(t, it), expIDs))
}

// TestMatchSearchWithOffset verifies the document search logic when searching
// for keyword matches and skipping some results.
func (s *SuiteBase) TestMatchSearchWithOffset(t *testing.T) {
	var (
		numDocs = 50
		expIDs  []uuid.UUID
	)
	for i := 0; i < numDocs; i++ {
		id := uuid.New()
		expIDs = append(expIDs, id)
		doc := &domain.Document{
			LinkID:  id,
			Title:   fmt.Sprintf("doc with ID %s", id.String()),
			Content: "Ovidius poeta in terra pontica",
		}

		err := s.idx.Index(doc)
		assert.Nil(t, err)

		err = s.idx.UpdateScore(id, float64(numDocs-i))
		assert.Nil(t, err)
	}

	it, err := s.idx.Search(&ports.DocumentQuery{
		Type:       ports.DocumentQueryTypeMatch,
		Expression: "poeta",
		Offset:     20,
	})
	assert.Nil(t, err)
	assert.True(t, reflect.DeepEqual(iterateDocs(t, it), expIDs[20:]))

	// Search with offset beyon the total number of results
	it, err = s.idx.Search(&ports.DocumentQuery{
		Type:       ports.DocumentQueryTypeMatch,
		Expression: "poeta",
		Offset:     200,
	})
	assert.Nil(t, err)
	assert.Len(t, iterateDocs(t, it), 0)
}

// TestUpdateScore checks that PageRank score updates work as expected.
func (s *SuiteBase) TestUpdateScore(t *testing.T) {
	var (
		numDocs = 100
		expIDs  []uuid.UUID
	)
	for i := 0; i < numDocs; i++ {
		id := uuid.New()
		expIDs = append(expIDs, id)
		doc := &domain.Document{
			LinkID:  id,
			Title:   fmt.Sprintf("doc with ID %s", id.String()),
			Content: "Ovidius poeta in terra pontica",
		}

		err := s.idx.Index(doc)
		assert.Nil(t, err)

		err = s.idx.UpdateScore(id, float64(numDocs-i))
		assert.Nil(t, err)
	}

	it, err := s.idx.Search(&ports.DocumentQuery{
		Type:       ports.DocumentQueryTypeMatch,
		Expression: "poeta",
	})
	assert.Nil(t, err)
	assert.Equal(t, iterateDocs(t, it), expIDs)

	// Update the pagerank scores so that results are sorted in the
	// reverse order.
	for i := 0; i < numDocs; i++ {
		err = s.idx.UpdateScore(expIDs[i], float64(i))
		assert.Nil(t, err, expIDs[i].String())
	}

	it, err = s.idx.Search(&ports.DocumentQuery{
		Type:       ports.DocumentQueryTypeMatch,
		Expression: "poeta",
	})
	assert.Nil(t, err)
	assert.True(t, reflect.DeepEqual(iterateDocs(t, it), reverse(expIDs)))
}

// TestUpdateScoreForUnknownDocument checks that a placeholder document will
// be created when setting the PageRank score for an unknown document.
func (s *SuiteBase) TestUpdateScoreForUnknownDocument(t *testing.T) {
	linkID := uuid.New()
	err := s.idx.UpdateScore(linkID, 0.5)
	assert.Nil(t, err)

	doc, err := s.idx.FindByID(linkID)
	assert.Nil(t, err)

	assert.Equal(t, doc.URL, "")
	assert.Equal(t, doc.Title, "")
	assert.Equal(t, doc.Content, "")
	assert.True(t, doc.IndexedAt.IsZero())
	assert.Equal(t, doc.PageRank, 0.5)
}

func iterateDocs(t *testing.T, it ports.DocumentIterator) []uuid.UUID {
	var seen []uuid.UUID
	for it.Next() {
		seen = append(seen, it.Document().LinkID)
	}
	assert.Nil(t, it.Error())
	assert.Nil(t, it.Close())
	return seen
}

func reverse(in []uuid.UUID) []uuid.UUID {
	for left, right := 0, len(in)-1; left < right; left, right = left+1, right-1 {
		in[left], in[right] = in[right], in[left]
	}

	return in
}
