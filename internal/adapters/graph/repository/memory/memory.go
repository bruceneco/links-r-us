package memory

import (
	"fmt"
	"github.com/bruceneco/links-r-us/internal/application/core/domain"
	"github.com/bruceneco/links-r-us/internal/ports/repository"
	"sync"
	"time"

	"github.com/google/uuid"
)

// edgeList contains the slice of edge UUIDs that originate from a link in the graph.
type edgeList []uuid.UUID

// InMemoryGraph implements an in-memory link graph that can be concurrently
// accessed by multiple clients.
type InMemoryGraph struct {
	mu sync.RWMutex

	links map[uuid.UUID]*domain.Link
	edges map[uuid.UUID]*domain.Edge

	linkURLIndex map[string]*domain.Link
	linkEdgeMap  map[uuid.UUID]edgeList
}

// NewInMemoryGraph creates a new in-memory link graph.
func NewInMemoryGraph() repository.GraphRepository {
	return &InMemoryGraph{
		links:        make(map[uuid.UUID]*domain.Link),
		edges:        make(map[uuid.UUID]*domain.Edge),
		linkURLIndex: make(map[string]*domain.Link),
		linkEdgeMap:  make(map[uuid.UUID]edgeList),
	}
}

// UpsertLink creates a new link or updates an existing link.
func (s *InMemoryGraph) UpsertLink(link *domain.Link) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if a link with the same URL already exists. If so, convert
	// this into an update and point the link ID to the existing link.
	if existing := s.linkURLIndex[link.URL]; existing != nil {
		link.ID = existing.ID
		origTs := existing.RetrievedAt
		*existing = *link
		if origTs.After(existing.RetrievedAt) {
			existing.RetrievedAt = origTs
		}
		return nil
	}

	// Assign new ID and insert link
	for {
		link.ID = uuid.New()
		if s.links[link.ID] == nil {
			break
		}
	}

	lCopy := new(domain.Link)
	*lCopy = *link
	s.linkURLIndex[lCopy.URL] = lCopy
	s.links[lCopy.ID] = lCopy
	return nil
}

// FindLink looks up a link by its ID.
func (s *InMemoryGraph) FindLink(id uuid.UUID) (*domain.Link, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	link := s.links[id]
	if link == nil {
		return nil, fmt.Errorf("find link: %w", repository.GraphErrNotFound)
	}

	lCopy := new(domain.Link)
	*lCopy = *link
	return lCopy, nil
}

// Links returns an iterator for the set of links whose IDs belong to the
// [fromID, toID) range and were retrieved before the provided timestamp.
func (s *InMemoryGraph) Links(fromID, toID uuid.UUID, retrievedBefore time.Time) (repository.LinkIterator, error) {
	from, to := fromID.String(), toID.String()

	s.mu.RLock()
	defer s.mu.RUnlock()
	var list []*domain.Link
	for linkID, link := range s.links {
		if id := linkID.String(); id >= from && id < to && link.RetrievedAt.Before(retrievedBefore) {
			linkCopy := new(domain.Link)
			*linkCopy = *link
			list = append(list, linkCopy)
		}
	}

	return &linkIterator{s: s, links: list}, nil
}

// UpsertEdge creates a new edge or updates an existing edge.
func (s *InMemoryGraph) UpsertEdge(edge *domain.Edge) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, srcExists := s.links[edge.Src]
	_, dstExists := s.links[edge.Dst]
	if !srcExists || !dstExists {
		return fmt.Errorf("upsert edge: %w", repository.GraphErrUnknownEdgeLinks)
	}

	// Scan edge list from source
	for _, edgeID := range s.linkEdgeMap[edge.Src] {
		existingEdge := s.edges[edgeID]
		if existingEdge.Dst == edge.Dst {
			existingEdge.UpdatedAt = time.Now()
			*edge = *existingEdge
			return nil
		}
	}

	// Insert new edge
	for {
		edge.ID = uuid.New()
		if s.edges[edge.ID] == nil {
			break
		}
	}

	edge.UpdatedAt = time.Now()
	eCopy := new(domain.Edge)
	*eCopy = *edge
	s.edges[eCopy.ID] = eCopy

	// Append the edge ID to the list of edges originating from the
	// edge's source link.
	s.linkEdgeMap[eCopy.Src] = append(s.linkEdgeMap[eCopy.Src], eCopy.ID)
	return nil
}

// Edges returns an iterator for the set of edges whose source vertex IDs
// belong to the [fromID, toID) range and were updated before the provided
// timestamp.
func (s *InMemoryGraph) Edges(fromID, toID uuid.UUID, updatedBefore time.Time) (repository.EdgeIterator, error) {
	from, to := fromID.String(), toID.String()

	s.mu.RLock()
	defer s.mu.RUnlock()
	var list []*domain.Edge
	for linkID := range s.links {
		if id := linkID.String(); id < from || id >= to {
			continue
		}

		for _, edgeID := range s.linkEdgeMap[linkID] {
			if edge := s.edges[edgeID]; edge.UpdatedAt.Before(updatedBefore) {
				edgeCopy := *edge
				list = append(list, &edgeCopy)
			}
		}
	}

	return &edgeIterator{s: s, edges: list}, nil
}

// RemoveStaleEdges removes any edge that originates from the specified link ID
// and was updated before the specified timestamp.
func (s *InMemoryGraph) RemoveStaleEdges(fromID uuid.UUID, updatedBefore time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var newEdgeList edgeList
	for _, edgeID := range s.linkEdgeMap[fromID] {
		edge := s.edges[edgeID]
		if edge.UpdatedAt.Before(updatedBefore) {
			delete(s.edges, edgeID)
			continue
		}

		newEdgeList = append(newEdgeList, edgeID)
	}

	// Replace edge list or origin link with the filtered edge list
	s.linkEdgeMap[fromID] = newEdgeList
	return nil
}
