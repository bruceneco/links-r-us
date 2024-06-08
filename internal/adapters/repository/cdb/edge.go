package cdb

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/bruceneco/links-r-us/internal/application/core/domain"
	"github.com/bruceneco/links-r-us/internal/ports"
	repoPorts "github.com/bruceneco/links-r-us/internal/ports/repository"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"time"
)

type EdgeRepository struct {
	db *sql.DB
}

func NewEdgeRepository(db *sql.DB) repoPorts.EdgeRepository {
	return &EdgeRepository{
		db: db,
	}
}

var upsertEdgeQuery = `
	INSERT INTO edges (src, dst, updated_at) VALUES ($1, $2, NOW())
	ON CONFLICT (src, dst) DO UPDATE SET updated_at = NOW()
	RETURNING id, updated_at
`

func (r *EdgeRepository) Upsert(edge *domain.Edge) error {
	row := r.db.QueryRow(upsertEdgeQuery, edge.Src, edge.Dst)
	if err := row.Scan(&edge.ID, &edge.UpdatedAt); err != nil {
		if isForeignKeyViolationError(err) {
			err = ports.GraphErrUnknownEdgeLinks
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

func (r *EdgeRepository) Edges(fromID, toID uuid.UUID, updatedBefore time.Time) (ports.EdgeIterator, error) {
	rows, err := r.db.Query(edgesInPartitionQuery, fromID, toID, updatedBefore)
	if err != nil {
		return nil, fmt.Errorf("edges: %w", err)
	}
	return newEdgeIterator(rows), nil
}

var removeStaleEdgesQuery = `
	DELETE FROM EDGES WHERE src=$1 AND updated_at < $2
`

func (r *EdgeRepository) RemoveStaleEdges(fromID uuid.UUID, updatedBefore time.Time) error {
	_, err := r.db.Exec(removeStaleEdgesQuery, fromID, updatedBefore.UTC())
	if err != nil {
		return fmt.Errorf("remove stale edges: %w", err)
	}
	return nil
}
