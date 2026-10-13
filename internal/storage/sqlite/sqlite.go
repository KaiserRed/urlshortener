package sqlite

import (
	"database/sql"
	"errors"
	"fmt"

	"url-shortener/internal/storage"

	"github.com/mattn/go-sqlite3"
)

type Storage struct {
	db *sql.DB
}

// New opens the SQLite database. The schema is managed by migrations.
func New(storagePath string) (*Storage, error) {
	const op = "storage.sqlite.New"

	db, err := sql.Open("sqlite3", storagePath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{db: db}, nil
}

// Close closes the underlying database connection.
func (s *Storage) Close() error {
	return s.db.Close()
}

// SaveURL inserts a new URL owned by userID and returns the new record ID.
func (s *Storage) SaveURL(urlToSave string, alias string, userID int64) (int64, error) {
	const op = "storage.sqlite.SaveURL"

	stmt, err := s.db.Prepare("INSERT INTO url(url, alias, user_id) VALUES(?, ?, ?)")
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	res, err := stmt.Exec(urlToSave, alias, userID)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			return 0, fmt.Errorf("%s: %w", op, storage.ErrUrlExists)
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("%s: failed to get last insert id: %w", op, err)
	}

	return id, nil
}

// GetURL returns the original URL by alias.
func (s *Storage) GetURL(alias string) (string, error) {
	const op = "storage.sqlite.GetURL"

	stmt, err := s.db.Prepare("SELECT url FROM url WHERE alias = ?")
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	var resURL string
	if err := stmt.QueryRow(alias).Scan(&resURL); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", storage.ErrURLNotFound
		}
		return "", fmt.Errorf("%s: execute statement: %w", op, err)
	}

	return resURL, nil
}

// URL returns the original URL and its owner ID by alias.
func (s *Storage) URL(alias string) (string, int64, error) {
	const op = "storage.sqlite.URL"

	stmt, err := s.db.Prepare("SELECT url, user_id FROM url WHERE alias = ?")
	if err != nil {
		return "", 0, fmt.Errorf("%s: %w", op, err)
	}

	var (
		resURL string
		userID int64
	)
	if err := stmt.QueryRow(alias).Scan(&resURL, &userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", 0, storage.ErrURLNotFound
		}
		return "", 0, fmt.Errorf("%s: execute statement: %w", op, err)
	}

	return resURL, userID, nil
}

// DeleteURL removes a URL by alias.
func (s *Storage) DeleteURL(alias string) error {
	const op = "storage.sqlite.DeleteURL"

	stmt, err := s.db.Prepare("DELETE FROM url WHERE alias = ?")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	res, err := stmt.Exec(alias)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if affected == 0 {
		return storage.ErrURLNotFound
	}

	return nil
}
