package storage

import (
	"context"
	"encoding/json"
)

// Store defines the interface for our key-value store
type Store interface {
	// Get retrieves a value for a given key
	Get(ctx context.Context, key string) ([]byte, error)

	// Set stores a value for a given key
	Set(ctx context.Context, key string, value []byte) error

	// Delete removes a key-value pair
	Delete(ctx context.Context, key string) error

	// List returns all keys with a given prefix
	List(ctx context.Context, prefix string) ([]string, error)

	// Close closes the store and releases any resources
	Close() error

	// Apply applies a command to the store
	Apply(cmd *Command) error

	// Snapshot returns a snapshot of the store
	Snapshot() ([]byte, error)

	// Restore restores the store from a snapshot
	Restore(snapshot []byte) error
}

// Command represents a command to be applied to the store
type Command struct {
	Op    string `json:"op"`
	Key   string `json:"key"`
	Value []byte `json:"value,omitempty"`
}

// Marshal marshals the command to JSON
func (c *Command) Marshal() ([]byte, error) {
	return json.Marshal(c)
}

// Unmarshal unmarshals the command from JSON
func (c *Command) Unmarshal(data []byte) error {
	return json.Unmarshal(data, c)
}

const (
	// Command operations
	OpSet    = "set"
	OpDelete = "delete"
)
