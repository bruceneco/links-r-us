package cdb

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/bruceneco/links-r-us/internal/application/core/domain"
	"github.com/bruceneco/links-r-us/internal/ports/repository"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"time"
)

type GraphCDBRepository struct {
	db *sql.DB
}

func NewGraphCDBRepository(db *sql.DB) repository.GraphRepository {
	return &GraphCDBRepository{
		db: db,
	}
}

var upsertEdgeQuery = `
	INSERT INTO edges (src, dst, updated_at) VALUES ($1, $2, NOW())
	ON CONFLICT (src, dst) DO UPDATE SET updated_at = NOW()
	RETURNING id, updated_at
`

func (r *GraphCDBRepository) UpsertEdge(edge *domain.Edge) error {
	row := r.db.QueryRow(upsertEdgeQuery, edge.Src, edge.Dst)
	if err := row.Scan(&edge.ID, &edge.UpdatedAt); err != nil {
		if isForeignKeyViolationError(err) {
			err = repository.GraphErrUnknownEdgeLinks
		}
		return fmt.Errorf("upsert edge: %w", err)
	}
	edge.UpdatedAt = edge.UpdatedAt.UTC()
	return nil
}

func isForeignKeyViolationError(err error) bool {
	var pqErr *pq.Error
	valid := errors.As(err, &pqErr)
	if !valid {
		return false
	}
	return pqErr.Code.Name() == "foreign_key_violation"
}

var edgesInPartitionQuery = `
	SELECT id, src, dst, updated_at 
	FROM edges 
	WHERE 
	    src >= $1 AND 
	    src < $2 AND 
	    updated_at < $3
`

func (r *GraphCDBRepository) Edges(fromID, toID uuid.UUID, updatedBefore time.Time) (repository.EdgeIterator, error) {
	rows, err := r.db.Query(edgesInPartitionQuery, fromID, toID, updatedBefore)
	if err != nil {
		return nil, fmt.Errorf("edges: %w", err)
	}
	return newEdgeIterator(rows), nil
}

var removeStaleEdgesQuery = `
	DELETE FROM edges WHERE src=$1 AND updated_at < $2
`

func (r *GraphCDBRepository) RemoveStaleEdges(fromID uuid.UUID, updatedBefore time.Time) error {
	_, err := r.db.Exec(removeStaleEdgesQuery, fromID, updatedBefore.UTC())
	if err != nil {
		return fmt.Errorf("remove stale edges: %w", err)
	}
	return nil
}

var upsertLinkQuery = `
	INSERT INTO links (url, retrieved_at) VALUES ($1, $2)
	ON CONFLICT (url) DO UPDATE SET retrieved_at=GREATEST(links.retrieved_at, $2)
	RETURNING id, retrieved_at
`

func (r *GraphCDBRepository) UpsertLink(link *domain.Link) error {
	row := r.db.QueryRow(upsertLinkQuery, link.URL, link.RetrievedAt.UTC())
	if err := row.Scan(&link.ID, &link.RetrievedAt); err != nil {
		return fmt.Errorf("upsert link: %w", err)
	}
	link.RetrievedAt = link.RetrievedAt.UTC()
	return nil
}

var findLinkQuery = `
	SELECT url, retrieved_at FROM links WHERE id=$1
`

func (r *GraphCDBRepository) FindLink(id uuid.UUID) (*domain.Link, error) {
	row := r.db.QueryRow(findLinkQuery, id)
	link := &domain.Link{ID: id}
	if err := row.Scan(&link.URL, &link.RetrievedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("find link: %w", repository.GraphErrNotFound)
		}
		return nil, fmt.Errorf("find link: %w", err)
	}
	link.RetrievedAt = link.RetrievedAt.UTC()
	return link, nil
}

var linksInPartitionQuery = `
	SELECT id, url, retrieved_at FROM links WHERE id >= $1 AND id <= $2 AND retrieved_at < $3
`

func (r *GraphCDBRepository) Links(fromId uuid.UUID, toId uuid.UUID, accessedBefore time.Time) (repository.LinkIterator, error) {
	rows, err := r.db.Query(linksInPartitionQuery, fromId, toId, accessedBefore.UTC())
	if err != nil {
		return nil, fmt.Errorf("links: %w", err)
	}
	return newLinkIterator(rows), nil
}
