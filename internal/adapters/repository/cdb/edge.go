package cdb

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/bruceneco/links-r-us/internal/application/core/domain"
	"github.com/bruceneco/links-r-us/internal/ports"
	repoPorts "github.com/bruceneco/links-r-us/internal/ports/repository"
	"github.com/lib/pq"
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

func (l *EdgeRepository) Upsert(edge *domain.Edge) error {
	row := l.db.QueryRow(upsertEdgeQuery, edge.Src, edge.Dst)
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
