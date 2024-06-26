package memory

import "github.com/bruceneco/links-r-us/internal/application/core/domain"

// linkIterator is a ports.LinkIterator implementation for the in-memory graph.
type linkIterator struct {
	s *InMemoryGraph

	links    []*domain.Link
	curIndex int
}

// Next implements ports.LinkIterator.
func (i *linkIterator) Next() bool {
	if i.curIndex >= len(i.links) {
		return false
	}
	i.curIndex++
	return true
}

// Error implements ports.LinkIterator.
func (i *linkIterator) Error() error {
	return nil
}

// Close implements ports.LinkIterator.
func (i *linkIterator) Close() error {
	return nil
}

// Link implements ports.LinkIterator.
func (i *linkIterator) Link() *domain.Link {
	// The link pointer contents may be overwritten by a graph update; to
	// avoid data-races we acquire the read lock first and clone the link
	i.s.mu.RLock()
	defer i.s.mu.RUnlock()
	link := *i.links[i.curIndex-1]
	return &link
}

// edgeIterator is a ports.EdgeIterator implementation for the in-memory graph.
type edgeIterator struct {
	s *InMemoryGraph

	edges    []*domain.Edge
	curIndex int
}

// Next implements ports.EdgeIterator.
func (i *edgeIterator) Next() bool {
	if i.curIndex >= len(i.edges) {
		return false
	}
	i.curIndex++
	return true
}

// Error implements graph.EdgeIterator.
func (i *edgeIterator) Error() error {
	return nil
}

// Close implements graph.EdgeIterator.
func (i *edgeIterator) Close() error {
	return nil
}

// Edge implements graph.EdgeIterator.
func (i *edgeIterator) Edge() *domain.Edge {
	// The edge pointer contents may be overwritten by a graph update; to
	// avoid data-races we acquire the read lock first and clone the edge
	i.s.mu.RLock()
	defer i.s.mu.RUnlock()
	edge := *i.edges[i.curIndex-1]
	return &edge
}
