package sqlite

import (
	"database/sql"
	"fmt"
	"github.com/mattn/go-sqlite3"
	_ "github.com/mattn/go-sqlite3" //sqlite3 driver
	"shorty/internal/storage"
	"time"
)

type Storage struct {
	db *sql.DB
}

const (
	OperationNew    = "storage.sqlite.New"
	OperationSave   = "storage.sqlite.SaveURL"
	OperationGet    = "storage.sqlite.GetURL"
	OperationDelete = "storage.sqlite.DeleteURL"
)

func New(dbPath string) (*Storage, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", OperationNew, err)
	}

	statement, err := db.Prepare(`
	CREATE TABLE IF NOT EXISTS url(
		id INTEGER PRIMARY KEY,
		alias TEXT NOT NULL UNIQUE,
		url TEXT NOT NULL,
		created_at TEXT NOT NULL
		);
	CREATE INDEX IF NOT EXISTS idx_alias ON url(alias);`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", OperationNew, err)
	}

	_, err = statement.Exec()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", OperationNew, err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) SaveURL(url string, alias string) (int64, error) {
	statement, err := s.db.Prepare(`INSERT INTO url(url, alias, created_at) VALUES(?, ?, ?)`)
	if err != nil {
		return 0, fmt.Errorf("%s: prepare statement: %w", OperationSave, err)
	}

	result, err := statement.Exec(url, alias, time.Now().UTC().Format(time.DateTime))
	if err != nil {
		//cast to internal sqlite type and check if constraint was violated
		if sqliteErr, ok := err.(sqlite3.Error); ok && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {

			//if an url was added with an alias that was previously saved, then we throw an error
			return 0, fmt.Errorf("%s: %w", OperationSave, storage.ErrURLAlreadyExists)
		}
		return 0, fmt.Errorf("%s: %w", OperationSave, err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("%s: failed to get last insert id %w", OperationSave, err)
	}

	return id, nil
}

func (s *Storage) GetURL(alias string) (string, error) {
	var resultURL string

	statement, err := s.db.Prepare(`SELECT url FROM url WHERE alias = ?`)
	if err != nil {
		return "", fmt.Errorf("%s: prepare statement: %w", OperationGet, err)
	}

	err = statement.QueryRow(alias).Scan(&resultURL)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", storage.ErrURLNotFound
		}
		return "", fmt.Errorf("%s: execute statement %w", OperationGet, err)
	}

	return resultURL, nil
}

func (s *Storage) DeleteURL(alias string) error {
	statement, err := s.db.Prepare(`DELETE FROM url WHERE alias = ?`)
	if err != nil {
		return fmt.Errorf("%s: prepare statement: %w", OperationDelete, err)
	}

	_, err = statement.Exec(alias)
	if err != nil {
		return fmt.Errorf("%s: execute statement %w", OperationDelete, err)
	}

	return nil
}
