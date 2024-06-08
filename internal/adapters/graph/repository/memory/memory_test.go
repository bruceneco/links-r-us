package memory

import (
	"github.com/bruceneco/links-r-us/internal/adapters/graph/graphtest"
	"github.com/stretchr/testify/suite"
	"testing"
)

func Test(t *testing.T) {
	suite.Run(t, new(InMemoryGraphTestSuite))
}

type InMemoryGraphTestSuite struct {
	suite.Suite
	base graphtest.SuiteBase
}

func (s *InMemoryGraphTestSuite) SetupTest() {
	s.base.SetGraph(NewInMemoryGraph())
}
func (s *InMemoryGraphTestSuite) TestUpsertLink() {
	s.base.TestUpsertLink(s.T())
}
func (s *InMemoryGraphTestSuite) TestFindLink() {
	s.base.TestFindLink(s.T())
}
func (s *InMemoryGraphTestSuite) TestConcurrentLinkIterators() {
	s.base.TestConcurrentLinkIterators(s.T())
}
func (s *InMemoryGraphTestSuite) TestLinkIteratorTimeFilter() {
	s.base.TestLinkIteratorTimeFilter(s.T())
}
func (s *InMemoryGraphTestSuite) TestPartitionedLinkIterators() {
	s.base.TestPartitionedLinkIterators(s.T())
}
func (s *InMemoryGraphTestSuite) TestUpsertEdge() {
	s.base.TestUpsertEdge(s.T())
}
func (s *InMemoryGraphTestSuite) TestConcurrentEdgeIterators() {
	s.base.TestConcurrentEdgeIterators(s.T())
}
func (s *InMemoryGraphTestSuite) TestEdgeIteratorTimeFilter() {
	s.base.TestEdgeIteratorTimeFilter(s.T())
}
func (s *InMemoryGraphTestSuite) TestPartitionedEdgeIterators() {
	s.base.TestPartitionedEdgeIterators(s.T())
}
func (s *InMemoryGraphTestSuite) TestRemoveStaleEdges() {
	s.base.TestRemoveStaleEdges(s.T())
}
