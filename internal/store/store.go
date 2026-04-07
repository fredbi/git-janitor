// SPDX-License-Identifier: Apache-2.0

package store

import "io"

// Bucket name constants.
const (
	BucketCache   = "cache"   // keyed by repo path, TTL-based RepoInfo cache
	BucketHistory = "history" // keyed by timestamp, append-only action log (future)
	BucketAlerts  = "alerts"  // keyed by check+repo, ack/snooze state (future)
)

// Store is a simple key-value store organized into named buckets.
//
// Implementations must be safe for concurrent use.
type Store interface {
	io.Closer

	// Get retrieves the value for the given key in the named bucket.
	// Returns nil, nil if the key does not exist.
	Get(bucket, key string) ([]byte, error)

	// Put stores a value under the given key in the named bucket.
	Put(bucket, key string, value []byte) error

	// Delete removes the key from the named bucket.
	// It is not an error to delete a non-existent key.
	Delete(bucket, key string) error

	// Scan iterates over all key-value pairs in the named bucket whose
	// keys start with the given prefix. Pairs are visited in sorted key order.
	// The callback receives copies of key and value that are safe to retain.
	// Return false from the callback to stop iteration early.
	Scan(bucket, prefix string, fn func(key, value []byte) bool) error
}
