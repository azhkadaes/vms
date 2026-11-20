package storageimplementation

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/google/uuid"

	"github.com/azhkadaes/vms/backend/internal/infrastructure/storage"
)

type localStorage struct{ dir string }

var _ storage.Storage = (*localStorage)(nil)

// NewLocalStorage keeps each object as one private file named by its key directly in dir,
// creating dir when it is missing and proving it writable before uploads are enabled.
func NewLocalStorage(dir string) (storage.Storage, error) {
	if dir == "" || !filepath.IsAbs(dir) {
		return nil, errors.New("upload storage directory must be an absolute path")
	}
	dir = filepath.Clean(dir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, errors.New("upload storage directory cannot be created")
	}
	probe, err := os.CreateTemp(dir, ".probe-*")
	if err != nil {
		return nil, errors.New("upload storage directory is not writable")
	}
	_ = probe.Close()
	if err := os.Remove(probe.Name()); err != nil {
		return nil, errors.New("upload storage directory does not allow removal")
	}
	return &localStorage{dir: dir}, nil
}

// Put writes a hidden partial file and renames it to its key only once every byte is synced,
// so a key never names a partly written object.
func (s *localStorage) Put(ctx context.Context, content []byte) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	id, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("generate storage key: %w", err)
	}
	partial, err := os.CreateTemp(s.dir, ".partial-*")
	if err != nil {
		return "", fmt.Errorf("create storage object: %w", err)
	}
	published := false
	defer func() {
		if !published {
			_ = os.Remove(partial.Name())
		}
	}()
	_, err = partial.Write(content)
	if err == nil {
		err = partial.Sync()
	}
	if closeErr := partial.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return "", fmt.Errorf("write storage object: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	key := id.String()
	if err := os.Rename(partial.Name(), filepath.Join(s.dir, key)); err != nil {
		return "", fmt.Errorf("publish storage object: %w", err)
	}
	published = true
	return key, nil
}

// Open reads only objects named by a key this adapter issues; any other string names nothing.
func (s *localStorage) Open(_ context.Context, key string) (io.ReadCloser, error) {
	path, ok := s.path(key)
	if !ok {
		return nil, storage.ErrNotFound
	}
	file, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("open storage object: %w", err)
	}
	return file, nil
}

// Delete removes the object; a key that was never issued is not found, a removed one is fine.
func (s *localStorage) Delete(_ context.Context, key string) error {
	path, ok := s.path(key)
	if !ok {
		return storage.ErrNotFound
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("delete storage object: %w", err)
	}
	return nil
}

// path accepts only canonical lower-case UUID keys, so no key can leave the directory.
func (s *localStorage) path(key string) (string, bool) {
	id, err := uuid.Parse(key)
	if err != nil || id.String() != key {
		return "", false
	}
	return filepath.Join(s.dir, key), true
}
