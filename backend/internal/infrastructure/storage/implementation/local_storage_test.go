package storageimplementation_test

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"

	"github.com/azhkadaes/vms/backend/internal/infrastructure/storage"
	implementation "github.com/azhkadaes/vms/backend/internal/infrastructure/storage/implementation"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
)

func newStorage(t *testing.T) (storage.Storage, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "uploads")
	files, err := implementation.NewLocalStorage(dir)
	if err != nil {
		t.Fatal(err)
	}
	return files, dir
}

func read(t *testing.T, files storage.Storage, key string) ([]byte, error) {
	t.Helper()
	object, err := files.Open(context.Background(), key)
	if err != nil {
		return nil, err
	}
	defer object.Close()
	return io.ReadAll(object)
}

func TestStoredBytesReopenUnderAServerGeneratedUUIDKey(t *testing.T) {
	files, _ := newStorage(t)
	first, err := files.Put(context.Background(), []byte("%PDF-1.7 synthetic"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := files.Put(context.Background(), []byte("%PDF-1.7 synthetic"))
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{first, second} {
		parsed, err := uuid.Parse(key)
		if err != nil || parsed.String() != key || parsed.Version() != 4 {
			t.Fatalf("key %q is not a canonical random UUID", key)
		}
	}
	if first == second {
		t.Fatal("two objects share one key")
	}
	got, err := read(t, files, first)
	if err != nil || string(got) != "%PDF-1.7 synthetic" {
		t.Fatalf("reopened %q, %v", got, err)
	}
}

func TestTheStorageDirectoryHoldsOnlyFinishedObjects(t *testing.T) {
	files, dir := newStorage(t)
	key, err := files.Put(context.Background(), []byte("object"))
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != key {
		t.Fatalf("directory holds %v, want only %s", entries, key)
	}
}

func TestDeletedObjectsAreGoneAndDeletingTwiceIsHarmless(t *testing.T) {
	files, _ := newStorage(t)
	key, err := files.Put(context.Background(), []byte("object"))
	if err != nil {
		t.Fatal(err)
	}
	if err := files.Delete(context.Background(), key); err != nil {
		t.Fatal(err)
	}
	if _, err := read(t, files, key); !errors.Is(err, storage.ErrNotFound) || !errors.Is(err, failure.ErrNotFound) {
		t.Fatalf("deleted object still opens: %v", err)
	}
	if err := files.Delete(context.Background(), key); err != nil {
		t.Fatalf("second delete: %v", err)
	}
}

// Keys come from the server, so anything else (a path, a URL, a client file name) names no object.
func TestKeysThatAreNotStorageKeysNameNoObject(t *testing.T) {
	files, dir := newStorage(t)
	outside := filepath.Join(filepath.Dir(dir), "outside.txt")
	if err := os.WriteFile(outside, []byte("private"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"", "../outside.txt", "..\\outside.txt", outside, "nota dinas.pdf", "https://example.test/x", "0F8FAD5B-D9CB-469F-A165-70867728950E"} {
		if _, err := read(t, files, key); !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("Open(%q) = %v, want not found", key, err)
		}
		if err := files.Delete(context.Background(), key); !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("Delete(%q) = %v, want not found", key, err)
		}
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatal("a delete reached outside the storage directory")
	}
}

func TestACanceledPutStoresNothing(t *testing.T) {
	files, dir := newStorage(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := files.Put(ctx, []byte("object")); !errors.Is(err, context.Canceled) {
		t.Fatalf("Put after cancel: %v", err)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Fatalf("canceled put left %v", entries)
	}
}

func TestStorageNeedsAnAbsoluteDirectory(t *testing.T) {
	if _, err := implementation.NewLocalStorage("relative/uploads"); err == nil {
		t.Fatal("relative storage directory accepted")
	}
	if _, err := implementation.NewLocalStorage(""); err == nil {
		t.Fatal("empty storage directory accepted")
	}
}
