package cdb

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/bruceneco/links-r-us/internal/application/core/domain"
	"github.com/bruceneco/links-r-us/internal/ports"
	repoPorts "github.com/bruceneco/links-r-us/internal/ports/repository"
	"github.com/google/uuid"
)

type LinkRepository struct {
	db *sql.DB
}

func NewLinkRepository(db *sql.DB) repoPorts.LinkRepository {
	return &LinkRepository{
		db: db,
	}
}

var upsertLinkQuery = `
	INSERT INTO links (url, retrieved_at) VALUES ($1, $2)
	ON CONFLICT (url) DO UPDATE SET retrieved_at=GREATEST(links.retrieved_at, $2)
	RETURNING id, retrieved_at
`

func (l *LinkRepository) Upsert(link *domain.Link) error {
	row := l.db.QueryRow(upsertLinkQuery, link.URL, link.RetrievedAt.UTC())
	if err := row.Scan(&link.ID, &link.RetrievedAt); err != nil {
		return fmt.Errorf("upsert link: %w", err)
	}
	link.RetrievedAt = link.RetrievedAt.UTC()
	return nil
}

var findLinkQuery = `
	SELECT url, retrieved_at FROM links WHERE id=$1
`

func (l *LinkRepository) Find(id uuid.UUID) (*domain.Link, error) {
	row := l.db.QueryRow(findLinkQuery, id)
	link := &domain.Link{ID: id}
	if err := row.Scan(&link.URL, &link.RetrievedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("find link: %w", ports.GraphErrNotFound)
		}
		return nil, fmt.Errorf("find link: %w", err)
	}
	link.RetrievedAt = link.RetrievedAt.UTC()
	return link, nil
}
