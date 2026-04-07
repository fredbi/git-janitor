// SPDX-License-Identifier: Apache-2.0

// Package bolt implements [store.Store] using bbolt (etcd-io/bbolt),
// a single-file B+ tree key-value database.
package bolt

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/fredbi/git-janitor/internal/config"
	"github.com/fredbi/git-janitor/internal/store"
	bolt "go.etcd.io/bbolt"
	bolterrors "go.etcd.io/bbolt/errors"
)

const (
	dbFile     = "janitor.db"
	dbFileMode = 0o600
	lockTimeout = 1 * time.Second
)

// ErrLocked is returned when the database file is already locked by
// another process (typically another git-janitor instance).
var ErrLocked = errors.New("database is locked — is another git-janitor instance running?")

var _ store.Store = (*Store)(nil)

// Store is a [store.Store] backed by a bbolt database file.
type Store struct {
	db *bolt.DB
}

// OpenDefault opens (or creates) the default bbolt database next to
// the configuration file (e.g. ~/.config/git-janitor/janitor.db).
// Returns nil, nil if the config directory cannot be determined.
func OpenDefault() (*Store, error) {
	cfgPath, err := config.DefaultConfigPath()
	if err != nil {
		return nil, nil //nolint:nilnil // no config dir = no store, not an error
	}

	dbPath := filepath.Join(filepath.Dir(cfgPath), dbFile)

	return New(dbPath)
}

// New opens (or creates) a bbolt database at the given path and
// ensures all required buckets exist.
//
// If the database is already locked by another process, New returns
// [ErrLocked] after a short timeout rather than blocking indefinitely.
func New(path string) (*Store, error) {
	db, err := bolt.Open(path, dbFileMode, &bolt.Options{Timeout: lockTimeout})
	if err != nil {
		// bbolt returns a generic "timeout" error when the file lock cannot be acquired.
		if errors.Is(err, bolterrors.ErrTimeout) {
			return nil, fmt.Errorf("%w: %s", ErrLocked, path)
		}

		return nil, fmt.Errorf("bolt: opening database %s: %w", path, err)
	}

	// Create all buckets upfront.
	err = db.Update(func(tx *bolt.Tx) error {
		for _, name := range []string{
			store.BucketCache,
			store.BucketHistory,
			store.BucketAlerts,
		} {
			if _, bucketErr := tx.CreateBucketIfNotExists([]byte(name)); bucketErr != nil {
				return fmt.Errorf("creating bucket %q: %w", name, bucketErr)
			}
		}

		return nil
	})
	if err != nil {
		_ = db.Close()

		return nil, fmt.Errorf("bolt: initializing buckets: %w", err)
	}

	return &Store{db: db}, nil
}

// Get retrieves the value for the given key in the named bucket.
// Returns nil, nil if the key does not exist.
func (s *Store) Get(bucket, key string) ([]byte, error) {
	var result []byte

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return nil
		}

		v := b.Get([]byte(key))
		if v != nil {
			// Copy the value: bbolt slices are only valid within the transaction.
			result = make([]byte, len(v))
			copy(result, v)
		}

		return nil
	})

	return result, err
}

// Put stores a value under the given key in the named bucket.
func (s *Store) Put(bucket, key string, value []byte) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return fmt.Errorf("bolt: bucket %q not found", bucket)
		}

		return b.Put([]byte(key), value)
	})
}

// Delete removes the key from the named bucket.
// It is not an error to delete a non-existent key.
func (s *Store) Delete(bucket, key string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return nil
		}

		return b.Delete([]byte(key))
	})
}

// Scan iterates over all key-value pairs whose keys start with the given prefix.
// Pairs are visited in sorted key order. The callback receives copies safe to retain.
func (s *Store) Scan(bucket, prefix string, fn func(key, value []byte) bool) error {
	return s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return nil
		}

		c := b.Cursor()
		pfx := []byte(prefix)

		for k, v := c.Seek(pfx); k != nil && len(k) >= len(pfx) && string(k[:len(pfx)]) == prefix; k, v = c.Next() {
			// Copy key and value: bbolt slices are only valid within the transaction.
			kCopy := make([]byte, len(k))
			copy(kCopy, k)

			vCopy := make([]byte, len(v))
			copy(vCopy, v)

			if !fn(kCopy, vCopy) {
				break
			}
		}

		return nil
	})
}

// Close closes the bbolt database.
func (s *Store) Close() error {
	return s.db.Close()
}
