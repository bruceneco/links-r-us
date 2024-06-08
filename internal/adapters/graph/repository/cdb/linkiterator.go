package cdb

import (
	"database/sql"
	"fmt"
	"github.com/bruceneco/links-r-us/internal/application/core/domain"
	"github.com/bruceneco/links-r-us/internal/ports/repository"
)

type linkIterator struct {
	rows        *sql.Rows
	lastErr     error
	latchedLink *domain.Link
}

func newLinkIterator(rows *sql.Rows) repository.LinkIterator {
	return &linkIterator{rows: rows}
}

func (i *linkIterator) Next() bool {
	if i.lastErr != nil || !i.rows.Next() {
		return false
	}

	link := new(domain.Link)
	i.lastErr = i.rows.Scan(&link.ID, &link.URL, &link.RetrievedAt)
	if i.lastErr != nil {
		return false
	}
	link.RetrievedAt = link.RetrievedAt.UTC()

	i.latchedLink = link
	return true
}

func (i *linkIterator) Error() error {
	return i.lastErr
}

func (i *linkIterator) Close() error {
	err := i.rows.Close()
	if err != nil {
		return fmt.Errorf("link iterator: %w", err)
	}
	return nil
}

func (i *linkIterator) Link() *domain.Link {
	return i.latchedLink
}
