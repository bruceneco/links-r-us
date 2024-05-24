package graphtest

import (
	"errors"
	"fmt"
	"github.com/bruceneco/links-r-us/internal/application/core/domain"
	"github.com/bruceneco/links-r-us/internal/ports"
	"github.com/stretchr/testify/assert"
	"math/big"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// SuiteBase defines a re-usable set of graph-related tests that can
// be executed against any type that implements ports.Graph.
type SuiteBase struct {
	g ports.Graph
}

// SetGraph configures the test-suite to run all tests against g.
func (s *SuiteBase) SetGraph(g ports.Graph) {
	s.g = g
}

// TestUpsertLink verifies the link upsert logic.
func (s *SuiteBase) TestUpsertLink(t *testing.T) {
	// Create a new link
	original := &domain.Link{
		URL:         "https://example.com",
		RetrievedAt: time.Now().Add(-10 * time.Hour),
	}

	err := s.g.UpsertLink(original)
	assert.Nil(t, err)
	assert.NotEqual(t, uuid.Nil, original.ID, "expected a linkID to be assigned to the new link")

	// Update existing link with a newer timestamp and different URL
	accessedAt := time.Now().Truncate(time.Second).UTC()
	existing := &domain.Link{
		ID:          original.ID,
		URL:         "https://example.com",
		RetrievedAt: accessedAt,
	}
	err = s.g.UpsertLink(existing)
	assert.Nil(t, err)
	assert.Equal(t, original.ID, existing.ID, "link ID changed while upserting")

	stored, err := s.g.FindLink(existing.ID)
	assert.Nil(t, err)
	assert.Equal(t, accessedAt, stored.RetrievedAt, "last accessed timestamp was not updated")

	// Attempt to insert a new link whose URL matches an existing link with
	// and provide an older accessedAt value
	sameURL := &domain.Link{
		URL:         existing.URL,
		RetrievedAt: time.Now().Add(-10 * time.Hour).UTC(),
	}
	err = s.g.UpsertLink(sameURL)
	assert.Nil(t, err)
	assert.Equal(t, existing.ID, sameURL.ID, "link ID changed while upserting")

	stored, err = s.g.FindLink(existing.ID)
	assert.Nil(t, err)
	assert.Equal(t, accessedAt, stored.RetrievedAt, "last accessed timestamp was overwritten with an older value")

	// Create a new link and then attempt to update its URL to the same as
	// an existing link.
	dup := &domain.Link{
		URL: "foo",
	}
	err = s.g.UpsertLink(dup)
	assert.Nil(t, err)
	assert.NotEqual(t, uuid.Nil, dup.ID, "expected a linkID to be assigned to the new link")
}

// TestFindLink verifies the link lookup logic.
func (s *SuiteBase) TestFindLink(t *testing.T) {
	// Create a new link
	link := &domain.Link{
		URL:         "https://example.com",
		RetrievedAt: time.Now().Truncate(time.Second).UTC(),
	}

	err := s.g.UpsertLink(link)
	assert.Nil(t, err)
	assert.NotEqual(t, uuid.Nil, link.ID, "expected a linkID to be assigned to the new link")

	// Lookup link by ID
	other, err := s.g.FindLink(link.ID)
	assert.Nil(t, err)

	assert.True(t, reflect.DeepEqual(link, other), "lookup by ID returned the wrong link")

	// Lookup link by unknown ID
	_, err = s.g.FindLink(uuid.Nil)
	assert.True(t, errors.Is(err, ports.GraphErrNotFound))
}

// TestConcurrentLinkIterators verifies that multiple clients can concurrently
// access the store.
func (s *SuiteBase) TestConcurrentLinkIterators(t *testing.T) {
	var (
		wg           sync.WaitGroup
		numIterators = 10
		numLinks     = 100
	)

	for i := 0; i < numLinks; i++ {
		link := &domain.Link{URL: fmt.Sprint(i)}
		assert.Nil(t, s.g.UpsertLink(link))
	}

	wg.Add(numIterators)
	for i := 0; i < numIterators; i++ {
		go func(id int) {
			defer wg.Done()

			itTagComment := fmt.Sprintf("iterator %d", id)
			seen := make(map[string]bool)
			it, err := s.partitionedLinkIterator(t, 0, 1, time.Now())
			assert.Nil(t, err, itTagComment)
			defer func() {
				assert.Nil(t, it.Close(), itTagComment)
			}()

			for i := 0; it.Next(); i++ {
				link := it.Link()
				linkID := link.ID.String()
				assert.Falsef(t, seen[linkID], "iterator %d saw same link twice", id)
				seen[linkID] = true
			}
			assert.Len(t, seen, numLinks, itTagComment)
			assert.Nil(t, it.Error(), itTagComment)
			assert.Nil(t, it.Close(), itTagComment)
		}(i)
	}

	doneCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneCh)
	}()

	select {
	case <-doneCh:
	// test completed successfully
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for test to complete")
	}
}

// TestLinkIteratorTimeFilter verifies that the time-based filtering of the
// link iterator works as expected.
func (s *SuiteBase) TestLinkIteratorTimeFilter(t *testing.T) {
	linkUUIDs := make([]uuid.UUID, 3)
	linkInsertTimes := make([]time.Time, len(linkUUIDs))
	for i := 0; i < len(linkUUIDs); i++ {
		link := &domain.Link{URL: fmt.Sprint(i), RetrievedAt: time.Now()}
		assert.Nil(t, s.g.UpsertLink(link))
		linkUUIDs[i] = link.ID
		linkInsertTimes[i] = time.Now()
	}

	for i, times := range linkInsertTimes {
		t.Logf("fetching links created before edge %d", i)
		s.assertIteratedLinkIDsMatch(t, times, linkUUIDs[:i+1])
	}
}

func (s *SuiteBase) assertIteratedLinkIDsMatch(t *testing.T, updatedBefore time.Time, exp []uuid.UUID) {
	it, err := s.partitionedLinkIterator(t, 0, 1, updatedBefore)
	assert.Nil(t, err)

	var got []uuid.UUID
	for it.Next() {
		got = append(got, it.Link().ID)
	}
	assert.Nil(t, it.Error())
	assert.Nil(t, it.Close())

	sort.Slice(got, func(l, r int) bool { return got[l].String() < got[r].String() })
	sort.Slice(exp, func(l, r int) bool { return exp[l].String() < exp[r].String() })
	assert.True(t, reflect.DeepEqual(got, exp))
}

// TestPartitionedLinkIterators verifies that the graph partitioning logic
// works as expected even when partitions contain an uneven number of items.
func (s *SuiteBase) TestPartitionedLinkIterators(t *testing.T) {
	numLinks := 100
	numPartitions := 10
	for i := 0; i < numLinks; i++ {
		assert.Nil(t, s.g.UpsertLink(&domain.Link{URL: fmt.Sprint(i)}))
	}

	// Check with both odd and even partition counts to check for rounding-related bugs.
	assert.Equal(t, numLinks, s.iteratePartitionedLinks(t, numPartitions))
	assert.Equal(t, numLinks, s.iteratePartitionedLinks(t, numPartitions+1))
}

func (s *SuiteBase) iteratePartitionedLinks(t *testing.T, numPartitions int) int {
	seen := make(map[string]bool)
	for partition := 0; partition < numPartitions; partition++ {
		it, err := s.partitionedLinkIterator(t, partition, numPartitions, time.Now())
		assert.Nil(t, err)
		defer func() {
			assert.Nil(t, it.Close())
		}()

		for it.Next() {
			link := it.Link()
			linkID := link.ID.String()
			assert.False(t, seen[linkID], "iterator returned same link in different partitions")
			seen[linkID] = true
		}
		assert.Nil(t, it.Error())
		assert.Nil(t, it.Close())
	}

	return len(seen)
}

// TestUpsertEdge verifies the edge upsert logic.
func (s *SuiteBase) TestUpsertEdge(t *testing.T) {
	// Create links
	linkUUIDs := make([]uuid.UUID, 3)
	for i := 0; i < 3; i++ {
		link := &domain.Link{URL: fmt.Sprint(i)}
		assert.Nil(t, s.g.UpsertLink(link))
		linkUUIDs[i] = link.ID
	}

	// Create a edge
	edge := &domain.Edge{
		Src: linkUUIDs[0],
		Dst: linkUUIDs[1],
	}

	err := s.g.UpsertEdge(edge)
	assert.Nil(t, err)
	assert.NotEqual(t, uuid.Nil, edge.ID, "expected an edgeID to be assigned to the new edge")
	assert.False(t, edge.UpdatedAt.IsZero(), "UpdatedAt field not set")

	// Update existing edge
	other := &domain.Edge{
		ID:  edge.ID,
		Src: linkUUIDs[0],
		Dst: linkUUIDs[1],
	}
	err = s.g.UpsertEdge(other)
	assert.Nil(t, err)
	assert.Equal(t, edge.ID, other.ID, "edge ID changed while upserting")
	assert.NotEqual(t, edge.UpdatedAt, other.UpdatedAt, "UpdatedAt field not modified")

	// Create edge with unknown link IDs
	bogus := &domain.Edge{
		Src: linkUUIDs[0],
		Dst: uuid.New(),
	}
	err = s.g.UpsertEdge(bogus)
	assert.True(t, errors.Is(err, ports.GraphErrUnknownEdgeLinks))
}

// TestConcurrentEdgeIterators verifies that multiple clients can concurrently
// access the store.
func (s *SuiteBase) TestConcurrentEdgeIterators(t *testing.T) {
	var (
		wg           sync.WaitGroup
		numIterators = 10
		numEdges     = 100
		linkUUIDs    = make([]uuid.UUID, numEdges*2)
	)

	for i := 0; i < numEdges*2; i++ {
		link := &domain.Link{URL: fmt.Sprint(i)}
		assert.Nil(t, s.g.UpsertLink(link))
		linkUUIDs[i] = link.ID
	}
	for i := 0; i < numEdges; i++ {
		assert.Nil(t, s.g.UpsertEdge(&domain.Edge{
			Src: linkUUIDs[0],
			Dst: linkUUIDs[i],
		}))
	}

	wg.Add(numIterators)
	for i := 0; i < numIterators; i++ {
		go func(id int) {
			defer wg.Done()

			itTagComment := fmt.Sprintf("iterator %d", id)
			seen := make(map[string]bool)
			it, err := s.partitionedEdgeIterator(t, 0, 1, time.Now())
			assert.Nil(t, err, itTagComment)
			defer func() {
				assert.Nil(t, it.Close(), itTagComment)
			}()

			for i := 0; it.Next(); i++ {
				edge := it.Edge()
				edgeID := edge.ID.String()
				assert.Falsef(t, seen[edgeID], "iterator %d saw same edge twice", id)
				seen[edgeID] = true
			}
			assert.Len(t, seen, numEdges, itTagComment)
			assert.Nil(t, it.Error(), itTagComment)
			assert.Nil(t, it.Close(), itTagComment)
		}(i)
	}

	doneCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneCh)
	}()

	select {
	case <-doneCh:
	// test completed successfully
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for test to complete")
	}
}

// TestEdgeIteratorTimeFilter verifies that the time-based filtering of the
// edge iterator works as expected.
func (s *SuiteBase) TestEdgeIteratorTimeFilter(t *testing.T) {
	linkUUIDs := make([]uuid.UUID, 3)
	linkInsertTimes := make([]time.Time, len(linkUUIDs))
	for i := 0; i < len(linkUUIDs); i++ {
		link := &domain.Link{URL: fmt.Sprint(i)}
		assert.Nil(t, s.g.UpsertLink(link))
		linkUUIDs[i] = link.ID
		linkInsertTimes[i] = time.Now()
	}

	edgeUUIDs := make([]uuid.UUID, len(linkUUIDs))
	edgeInsertTimes := make([]time.Time, len(linkUUIDs))
	for i := 0; i < len(linkUUIDs); i++ {
		edge := &domain.Edge{Src: linkUUIDs[0], Dst: linkUUIDs[i]}
		assert.Nil(t, s.g.UpsertEdge(edge))
		edgeUUIDs[i] = edge.ID
		edgeInsertTimes[i] = time.Now()
	}

	for i, times := range edgeInsertTimes {
		t.Logf("fetching edges created before edge %d", i)
		s.assertIteratedEdgeIDsMatch(t, times, edgeUUIDs[:i+1])
	}
}

func (s *SuiteBase) assertIteratedEdgeIDsMatch(t *testing.T, updatedBefore time.Time, exp []uuid.UUID) {
	it, err := s.partitionedEdgeIterator(t, 0, 1, updatedBefore)
	assert.Nil(t, err)

	var got []uuid.UUID
	for it.Next() {
		got = append(got, it.Edge().ID)
	}
	assert.Nil(t, it.Error())
	assert.Nil(t, it.Close())

	sort.Slice(got, func(l, r int) bool { return got[l].String() < got[r].String() })
	sort.Slice(exp, func(l, r int) bool { return exp[l].String() < exp[r].String() })
	assert.True(t, reflect.DeepEqual(got, exp))
}

// TestPartitionedEdgeIterators verifies that the graph partitioning logic
// works as expected even when partitions contain an uneven number of items.
func (s *SuiteBase) TestPartitionedEdgeIterators(t *testing.T) {
	numEdges := 100
	numPartitions := 10
	linkUUIDs := make([]uuid.UUID, numEdges*2)
	for i := 0; i < numEdges*2; i++ {
		link := &domain.Link{URL: fmt.Sprint(i)}
		assert.Nil(t, s.g.UpsertLink(link))
		linkUUIDs[i] = link.ID
	}
	for i := 0; i < numEdges; i++ {
		assert.Nil(t, s.g.UpsertEdge(&domain.Edge{
			Src: linkUUIDs[0],
			Dst: linkUUIDs[i],
		}))
	}

	// Check with both odd and even partition counts to check for rounding-related bugs.
	assert.Equal(t, numEdges, s.iteratePartitionedEdges(t, numPartitions))
	assert.Equal(t, numEdges, s.iteratePartitionedEdges(t, numPartitions+1))
}

func (s *SuiteBase) iteratePartitionedEdges(t *testing.T, numPartitions int) int {
	seen := make(map[string]bool)
	for partition := 0; partition < numPartitions; partition++ {
		// Build list of expected edges per partition. An edge belongs to a
		// partition if its origin link also belongs to the same partition.
		linksInPartition := make(map[uuid.UUID]struct{})
		linkIt, err := s.partitionedLinkIterator(t, partition, numPartitions, time.Now())
		assert.Nil(t, err)
		for linkIt.Next() {
			linkID := linkIt.Link().ID
			linksInPartition[linkID] = struct{}{}
		}

		it, err := s.partitionedEdgeIterator(t, partition, numPartitions, time.Now())
		assert.Nil(t, err)
		defer func() {
			assert.Nil(t, it.Close())
		}()

		for it.Next() {
			edge := it.Edge()
			edgeID := edge.ID.String()
			assert.False(t, seen[edgeID], "iterator returned same edge in different partitions")
			seen[edgeID] = true

			_, srcInPartition := linksInPartition[edge.Src]
			assert.True(t, srcInPartition, "iterator returned an edge whose source link belongs to a different partition")
		}
		assert.Nil(t, it.Error())
		assert.Nil(t, it.Close())
	}

	return len(seen)
}

// TestRemoveStaleEdges verifies that the edge deletion logic works as expected.
func (s *SuiteBase) TestRemoveStaleEdges(t *testing.T) {
	numEdges := 100
	linkUUIDs := make([]uuid.UUID, numEdges*4)
	goneUUIDs := make(map[uuid.UUID]struct{})
	for i := 0; i < numEdges*4; i++ {
		link := &domain.Link{URL: fmt.Sprint(i)}
		assert.Nil(t, s.g.UpsertLink(link))
		linkUUIDs[i] = link.ID
	}

	var lastTs time.Time
	for i := 0; i < numEdges; i++ {
		e1 := &domain.Edge{
			Src: linkUUIDs[0],
			Dst: linkUUIDs[i],
		}
		assert.Nil(t, s.g.UpsertEdge(e1))
		goneUUIDs[e1.ID] = struct{}{}
		lastTs = e1.UpdatedAt
	}

	deleteBefore := lastTs.Add(time.Millisecond)
	time.Sleep(250 * time.Millisecond)

	// The following edges will have an updated at value > lastTs
	for i := 0; i < numEdges; i++ {
		e2 := &domain.Edge{
			Src: linkUUIDs[0],
			Dst: linkUUIDs[numEdges+i+1],
		}
		assert.Nil(t, s.g.UpsertEdge(e2))
	}
	assert.Nil(t, s.g.RemoveStaleEdges(linkUUIDs[0], deleteBefore))

	it, err := s.partitionedEdgeIterator(t, 0, 1, time.Now())
	assert.Nil(t, err)
	defer func() { assert.Nil(t, it.Close()) }()

	var seen int
	for it.Next() {
		id := it.Edge().ID
		_, found := goneUUIDs[id]
		assert.Falsef(t, found, "expected edge %s to be removed from the edge list", id.String())
		seen++
	}

	assert.Equal(t, numEdges, seen)
}

func (s *SuiteBase) partitionedLinkIterator(t *testing.T, partition, numPartitions int, accessedBefore time.Time) (ports.LinkIterator, error) {
	from, to := s.partitionRange(t, partition, numPartitions)
	return s.g.Links(from, to, accessedBefore)
}

func (s *SuiteBase) partitionedEdgeIterator(t *testing.T, partition, numPartitions int, updatedBefore time.Time) (ports.EdgeIterator, error) {
	from, to := s.partitionRange(t, partition, numPartitions)
	return s.g.Edges(from, to, updatedBefore)
}

func (s *SuiteBase) partitionRange(t *testing.T, partition, numPartitions int) (from, to uuid.UUID) {
	if partition < 0 || partition >= numPartitions {
		t.Fatal("invalid partition")
	}

	var minUUID = uuid.Nil
	var maxUUID = uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")
	var err error

	// Calculate the size of each partition as: (2^128 / numPartitions)
	tokenRange := big.NewInt(0)
	partSize := big.NewInt(0)
	partSize.SetBytes(maxUUID[:])
	partSize = partSize.Div(partSize, big.NewInt(int64(numPartitions)))

	// We model the partitions as a segment that begins at minUUID (all
	// bits set to zero) and ends at maxUUID (all bits set to 1). By
	// setting the end range for the *last* partition to maxUUID we ensure
	// that we always cover the full range of UUIDs even if the range
	// itself is not evenly divisible by numPartitions.
	if partition == 0 {
		from = minUUID
	} else {
		tokenRange.Mul(partSize, big.NewInt(int64(partition)))
		from, err = uuid.FromBytes(tokenRange.Bytes())
		assert.Nil(t, err)
	}

	if partition == numPartitions-1 {
		to = maxUUID
	} else {
		tokenRange.Mul(partSize, big.NewInt(int64(partition+1)))
		to, err = uuid.FromBytes(tokenRange.Bytes())
		assert.Nil(t, err)
	}

	return from, to
}
