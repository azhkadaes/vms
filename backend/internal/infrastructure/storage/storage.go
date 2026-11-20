// Package storage is the private file storage adapter (D15). Keys are server-generated
// and never appear in responses, URLs or client-supplied paths.
package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
)

// ErrNotFound marks a key that names no stored object, including any key the adapter never issued.
var ErrNotFound = fmt.Errorf("storage: %w", failure.ErrNotFound)

// Storage keeps uploaded bytes privately under keys it generates itself.
type Storage interface {
	// Put stores content under a new random UUID key and returns the key once the bytes are written.
	Put(ctx context.Context, content []byte) (string, error)
	// Open reads the object stored under key, or returns ErrNotFound.
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	// Delete removes the object stored under key; a key already removed is not an error.
	Delete(ctx context.Context, key string) error
}
