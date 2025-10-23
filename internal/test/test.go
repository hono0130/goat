package test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

)

func FixtureDir(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(filename), "..", "..", "testdata")
}

func ReadGolden(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(FixtureDir(t), name)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read golden file %s: %v", path, err)
	}
	return string(b)
}
