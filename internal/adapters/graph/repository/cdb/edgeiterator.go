package cdb

import (
	"database/sql"
	"fmt"
	"github.com/bruceneco/links-r-us/internal/application/core/domain"
	"github.com/bruceneco/links-r-us/internal/ports"
)

type edgeIterator struct {
	rows        *sql.Rows
	lastErr     error
	latchedEdge *domain.Edge
}

func newEdgeIterator(rows *sql.Rows) ports.EdgeIterator {
	return &edgeIterator{rows: rows}
}

func (i *edgeIterator) Next() bool {
	if i.lastErr != nil || !i.rows.Next() {
		return false
	}

	edge := new(domain.Edge)
	i.lastErr = i.rows.Scan(&edge.ID, &edge.Src, &edge.Dst, &edge.UpdatedAt)
	if i.lastErr != nil {
		return false
	}
	edge.UpdatedAt = edge.UpdatedAt.UTC()

	i.latchedEdge = edge
	return true
}

func (i *edgeIterator) Error() error {
	return i.lastErr
}

func (i *edgeIterator) Close() error {
	err := i.rows.Close()
	if err != nil {
		return fmt.Errorf("edge iterator: %w", err)
	}
	return nil
}

func (i *edgeIterator) Edge() *domain.Edge {
	return i.latchedEdge
}
