package mermaid

import (
	"path/filepath"
	"runtime"
	"testing"
)

func workflowDir(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(filename), "testdata", "workflow")
}

func loadWorkflowPackage(t *testing.T) *packageInfo {
	t.Helper()
	pkg, err := loadPackageWithTypes(workflowDir(t))
	if err != nil {
		t.Fatalf("loadPackageWithTypes returned error: %v", err)
	}
	return pkg
}
