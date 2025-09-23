package mermaid

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPackageWithTypes(t *testing.T) {
	t.Setenv("GOCACHE", t.TempDir())

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}

	moduleRoot, err := findModuleRoot(wd)
	if err != nil {
		t.Fatalf("findModuleRoot returned error: %v", err)
	}

	dir, err := os.MkdirTemp(moduleRoot, "withcmp")
	if err != nil {
		t.Fatalf("MkdirTemp returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})

	source := "package withcmp\n\n" +
		"import \"github.com/google/go-cmp/cmp\"\n\n" +
		"var _ = cmp.Diff\n"
	if err := os.WriteFile(filepath.Join(dir, "withcmp.go"), []byte(source), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	pkg, err := loadPackageWithTypes(dir)
	if err != nil {
		t.Fatalf("loadPackageWithTypes returned error: %v", err)
	}
	if pkg == nil {
		t.Fatal("loadPackageWithTypes returned nil packageInfo")
	}
	if pkg.TypesInfo == nil {
		t.Fatal("TypesInfo should not be nil")
	}

	found := false
	for ident, obj := range pkg.TypesInfo.Uses {
		if ident == nil || obj == nil {
			continue
		}
		if ident.Name == "Diff" && obj.Pkg() != nil && obj.Pkg().Path() == "github.com/google/go-cmp/cmp" {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("expected to resolve cmp.Diff symbol")
	}
}
