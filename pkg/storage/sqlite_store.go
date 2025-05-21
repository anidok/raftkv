package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStore implements the Store interface using SQLite
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates a new SQLite store
func NewSQLiteStore(dataDir string) (*SQLiteStore, error) {
	dbPath := filepath.Join(dataDir, "raftkv.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite db: %w", err)
	}

	// Create the table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS kv (
			key TEXT PRIMARY KEY,
			value BLOB
		)
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

// Get retrieves a value for a given key
func (s *SQLiteStore) Get(ctx context.Context, key string) ([]byte, error) {
	var value []byte
	err := s.db.QueryRowContext(ctx, "SELECT value FROM kv WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return value, err
}

// Set stores a value for a given key
func (s *SQLiteStore) Set(ctx context.Context, key string, value []byte) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT OR REPLACE INTO kv (key, value) VALUES (?, ?)",
		key, value,
	)
	return err
}

// Delete removes a key-value pair
func (s *SQLiteStore) Delete(ctx context.Context, key string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM kv WHERE key = ?", key)
	return err
}

// List returns all keys with a given prefix
func (s *SQLiteStore) List(ctx context.Context, prefix string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT key FROM kv WHERE key LIKE ?", prefix+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, rows.Err()
}

// Close closes the store and releases any resources
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// Apply applies a command to the store
func (s *SQLiteStore) Apply(cmd *Command) error {
	switch cmd.Op {
	case OpSet:
		return s.Set(context.Background(), cmd.Key, cmd.Value)
	case OpDelete:
		return s.Delete(context.Background(), cmd.Key)
	default:
		return fmt.Errorf("unknown command operation: %s", cmd.Op)
	}
}

// Snapshot returns a snapshot of the store
func (s *SQLiteStore) Snapshot() ([]byte, error) {
	rows, err := s.db.Query("SELECT key, value FROM kv")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	data := make(map[string][]byte)
	for rows.Next() {
		var key string
		var value []byte
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		data[key] = value
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return json.Marshal(data)
}

// Restore restores the store from a snapshot
func (s *SQLiteStore) Restore(snapshot []byte) error {
	var data map[string][]byte
	if err := json.Unmarshal(snapshot, &data); err != nil {
		return err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Clear existing data
	_, err = tx.Exec("DELETE FROM kv")
	if err != nil {
		return err
	}

	// Restore from snapshot
	stmt, err := tx.Prepare("INSERT INTO kv (key, value) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for k, v := range data {
		if _, err := stmt.Exec(k, v); err != nil {
			return err
		}
	}

	return tx.Commit()
}
