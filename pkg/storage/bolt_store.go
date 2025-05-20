package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/boltdb/bolt"
)

// BoltStore implements the Store interface using BoltDB
type BoltStore struct {
	db *bolt.DB
}

// NewBoltStore creates a new BoltDB store
func NewBoltStore(dataDir string) (*BoltStore, error) {
	dbPath := filepath.Join(dataDir, "raftkv.db")
	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open bolt db: %w", err)
	}

	// Create the bucket if it doesn't exist
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("kv"))
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create bucket: %w", err)
	}

	return &BoltStore{db: db}, nil
}

// Get retrieves a value for a given key
func (s *BoltStore) Get(ctx context.Context, key string) ([]byte, error) {
	var value []byte
	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("kv"))
		if bucket == nil {
			return fmt.Errorf("bucket not found")
		}
		value = bucket.Get([]byte(key))
		return nil
	})
	return value, err
}

// Set stores a value for a given key
func (s *BoltStore) Set(ctx context.Context, key string, value []byte) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("kv"))
		if bucket == nil {
			return fmt.Errorf("bucket not found")
		}
		return bucket.Put([]byte(key), value)
	})
}

// Delete removes a key-value pair
func (s *BoltStore) Delete(ctx context.Context, key string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("kv"))
		if bucket == nil {
			return fmt.Errorf("bucket not found")
		}
		return bucket.Delete([]byte(key))
	})
}

// List returns all keys with a given prefix
func (s *BoltStore) List(ctx context.Context, prefix string) ([]string, error) {
	var keys []string
	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("kv"))
		if bucket == nil {
			return fmt.Errorf("bucket not found")
		}

		cursor := bucket.Cursor()
		prefixBytes := []byte(prefix)
		for k, _ := cursor.Seek(prefixBytes); k != nil && string(k)[:len(prefix)] == prefix; k, _ = cursor.Next() {
			keys = append(keys, string(k))
		}
		return nil
	})
	return keys, err
}

// Close closes the store and releases any resources
func (s *BoltStore) Close() error {
	return s.db.Close()
}

// Apply applies a command to the store
func (s *BoltStore) Apply(cmd *Command) error {
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
func (s *BoltStore) Snapshot() ([]byte, error) {
	var snapshot []byte
	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("kv"))
		if bucket == nil {
			return fmt.Errorf("bucket not found")
		}

		data := make(map[string][]byte)
		cursor := bucket.Cursor()
		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			data[string(k)] = v
		}

		var err error
		snapshot, err = json.Marshal(data)
		return err
	})
	return snapshot, err
}

// Restore restores the store from a snapshot
func (s *BoltStore) Restore(snapshot []byte) error {
	var data map[string][]byte
	if err := json.Unmarshal(snapshot, &data); err != nil {
		return err
	}

	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("kv"))
		if bucket == nil {
			return fmt.Errorf("bucket not found")
		}

		// Clear existing data
		if err := bucket.ForEach(func(k, _ []byte) error {
			return bucket.Delete(k)
		}); err != nil {
			return err
		}

		// Restore from snapshot
		for k, v := range data {
			if err := bucket.Put([]byte(k), v); err != nil {
				return err
			}
		}
		return nil
	})
}
