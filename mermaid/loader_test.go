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

func TestFindModuleRoot(t *testing.T) {
	t.Run("found go.mod ancestor", func(t *testing.T) {
		moduleRoot := t.TempDir()
		if err := os.WriteFile(filepath.Join(moduleRoot, "go.mod"), []byte("module example.com/mod\n"), 0o600); err != nil {
			t.Fatalf("WriteFile returned error: %v", err)
		}

		nested := filepath.Join(moduleRoot, "nested", "package")
		if err := os.MkdirAll(nested, 0o750); err != nil {
			t.Fatalf("MkdirAll returned error: %v", err)
		}

		got, err := findModuleRoot(nested)
		if err != nil {
			t.Fatalf("findModuleRoot returned error: %v", err)
		}
		if filepath.Clean(got) != filepath.Clean(moduleRoot) {
			t.Fatalf("findModuleRoot = %q, want %q", got, moduleRoot)
		}
	})

	t.Run("go.mod not found", func(t *testing.T) {
		start := t.TempDir()
		nested := filepath.Join(start, "missing")
		if err := os.MkdirAll(nested, 0o750); err != nil {
			t.Fatalf("MkdirAll returned error: %v", err)
		}

		if _, err := findModuleRoot(nested); err == nil {
			t.Fatal("findModuleRoot returned nil error for missing go.mod")
		}
	})
}

func TestReadModulePath(t *testing.T) {
	t.Run("reads module directive", func(t *testing.T) {
		moduleRoot := t.TempDir()
		content := "module example.com/hello\n"
		if err := os.WriteFile(filepath.Join(moduleRoot, "go.mod"), []byte(content), 0o600); err != nil {
			t.Fatalf("WriteFile returned error: %v", err)
		}

		got, err := readModulePath(moduleRoot)
		if err != nil {
			t.Fatalf("readModulePath returned error: %v", err)
		}
		const want = "example.com/hello"
		if got != want {
			t.Fatalf("readModulePath = %q, want %q", got, want)
		}
	})

	t.Run("missing module directive", func(t *testing.T) {
		moduleRoot := t.TempDir()
		if err := os.WriteFile(filepath.Join(moduleRoot, "go.mod"), []byte("// no module directive\n"), 0o600); err != nil {
			t.Fatalf("WriteFile returned error: %v", err)
		}

		if _, err := readModulePath(moduleRoot); err == nil {
			t.Fatal("readModulePath returned nil error for missing module directive")
		}
	})
}

func TestPackageImportPath(t *testing.T) {
	t.Run("module root", func(t *testing.T) {
		moduleRoot := t.TempDir()
		got, err := packageImportPath(moduleRoot, "example.com/mod", moduleRoot)
		if err != nil {
			t.Fatalf("packageImportPath returned error: %v", err)
		}
		if got != "example.com/mod" {
			t.Fatalf("packageImportPath = %q, want %q", got, "example.com/mod")
		}
	})

	t.Run("sub directory", func(t *testing.T) {
		moduleRoot := t.TempDir()
		sub := filepath.Join(moduleRoot, "internal", "pkg")
		if err := os.MkdirAll(sub, 0o750); err != nil {
			t.Fatalf("MkdirAll returned error: %v", err)
		}

		got, err := packageImportPath(moduleRoot, "example.com/mod", sub)
		if err != nil {
			t.Fatalf("packageImportPath returned error: %v", err)
		}
		if got != "example.com/mod/internal/pkg" {
			t.Fatalf("packageImportPath = %q, want %q", got, "example.com/mod/internal/pkg")
		}
	})

	t.Run("outside module", func(t *testing.T) {
		moduleRoot := t.TempDir()
		other := t.TempDir()
		if _, err := packageImportPath(moduleRoot, "example.com/mod", other); err == nil {
			t.Fatal("packageImportPath returned nil error for directory outside module root")
		}
	})
}

func TestParseGoFiles(t *testing.T) {
	t.Run("valid file", func(t *testing.T) {
		dir := t.TempDir()
		const source = "package sample\n\nfunc Answer() int { return 42 }\n"
		if err := os.WriteFile(filepath.Join(dir, "sample.go"), []byte(source), 0o600); err != nil {
			t.Fatalf("WriteFile returned error: %v", err)
		}

		files, fset, err := parseGoFiles(dir, []string{"sample.go"})
		if err != nil {
			t.Fatalf("parseGoFiles returned error: %v", err)
		}
		if len(files) != 1 {
			t.Fatalf("parseGoFiles returned %d files, want 1", len(files))
		}
		if fset == nil {
			t.Fatal("parseGoFiles returned nil FileSet")
		}
	})

	t.Run("parse error", func(t *testing.T) {
		dir := t.TempDir()
		const source = "package sample\n\nfunc broken(\n"
		if err := os.WriteFile(filepath.Join(dir, "broken.go"), []byte(source), 0o600); err != nil {
			t.Fatalf("WriteFile returned error: %v", err)
		}

		if _, _, err := parseGoFiles(dir, []string{"broken.go"}); err == nil {
			t.Fatal("parseGoFiles returned nil error for invalid Go source")
		}
	})
}

func TestIsStandardLibraryPackage(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "stdlib", path: "fmt", want: true},
		{name: "third party", path: "github.com/google/go-cmp/cmp", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isStandardLibraryPackage(tt.path); got != tt.want {
				t.Fatalf("isStandardLibraryPackage(%q) = %t, want %t", tt.path, got, tt.want)
			}
		})
	}
}
