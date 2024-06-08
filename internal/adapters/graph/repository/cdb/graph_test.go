package cdb

import (
	"database/sql"
	"fmt"
	"github.com/bruceneco/links-r-us/internal/adapters/graph/graphtest"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/suite"
	"os"
	"testing"
)

func Test(t *testing.T) {
	suite.Run(t, new(GraphCDBRepositoryTestSuite))
}

type GraphCDBRepositoryTestSuite struct {
	suite.Suite
	base graphtest.SuiteBase
	db   *sql.DB
}

func (s *GraphCDBRepositoryTestSuite) SetupSuite() {
	err := godotenv.Load("../../../../../configs/.env")
	if err != nil {
		s.FailNow("cant load env")
	}
	host := os.Getenv("CDB_HOST")
	port := os.Getenv("CDB_PORT")
	user := os.Getenv("CDB_USER")
	database := os.Getenv("CDB_DATABASE")
	dsn := fmt.Sprintf("postgres://%s@%s:%s/%s?sslmode=disable", user, host, port, database)
	fmt.Println(dsn)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		s.FailNowf("", "failed to connect to database %w", err)
	}
	err = db.Ping()
	if err != nil {
		s.FailNowf("", "failed to ping database %w", err)
	}
	r := NewGraphCDBRepository(db)
	s.db = db
	s.base.SetGraph(r)
}
func (s *GraphCDBRepositoryTestSuite) SetupTest() {
	s.flushDB()
}

func (s *GraphCDBRepositoryTestSuite) flushDB() {
	_, err := s.db.Exec("DELETE FROM links")
	s.Nil(err)
	_, err = s.db.Exec("DELETE FROM edges")
	s.Nil(err)
}
func (s *GraphCDBRepositoryTestSuite) TearDownSuite() {
	if s.db != nil {
		s.flushDB()
		s.Nil(s.db.Close())
	}
}
func (s *GraphCDBRepositoryTestSuite) TestUpsertLink() {
	s.base.TestUpsertLink(s.T())
}
func (s *GraphCDBRepositoryTestSuite) TestFindLink() {
	s.base.TestFindLink(s.T())
}
func (s *GraphCDBRepositoryTestSuite) TestConcurrentLinkIterators() {
	s.base.TestConcurrentLinkIterators(s.T())
}
func (s *GraphCDBRepositoryTestSuite) TestLinkIteratorTimeFilter() {
	s.base.TestLinkIteratorTimeFilter(s.T())
}
func (s *GraphCDBRepositoryTestSuite) TestPartitionedLinkIterators() {
	s.base.TestPartitionedLinkIterators(s.T())
}
func (s *GraphCDBRepositoryTestSuite) TestUpsertEdge() {
	s.base.TestUpsertEdge(s.T())
}
