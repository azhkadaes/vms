package pkg_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const internalPrefix = "github.com/azhkadaes/vms/backend/internal"

// Shared helpers depend only on the standard library, third parties and other pkg packages.
func TestNoPkgPackageImportsInternal(t *testing.T) {
	files := 0
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		files++
		// ImportsOnly ignores build tags, so integration-only files are checked too.
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, spec := range file.Imports {
			imported, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return err
			}
			if imported == internalPrefix || strings.HasPrefix(imported, internalPrefix+"/") {
				t.Errorf("%s imports %s", filepath.ToSlash(path), imported)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if files < 10 {
		t.Fatalf("walked only %d files; the check is not looking at pkg", files)
	}
}
