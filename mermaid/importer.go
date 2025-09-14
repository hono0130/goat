package mermaid

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// PackageInfo holds the parsed syntax and type information for a Go package.
type PackageInfo struct {
	Fset      *token.FileSet
	Syntax    []*ast.File
	TypesInfo *types.Info
}

type moduleImporter struct {
	root string
	std  types.Importer
}

func newModuleImporter(root string) moduleImporter {
	return moduleImporter{root: root, std: importer.Default()}
}

func (m moduleImporter) Import(path string) (*types.Package, error) {
	if path == goatPackageFullPath {
		return m.importGoat()
	}
	return m.std.Import(path)
}

func (m moduleImporter) ImportFrom(path, dir string, mode types.ImportMode) (*types.Package, error) {
	if path == goatPackageFullPath {
		return m.importGoat()
	}
	if imp, ok := m.std.(types.ImporterFrom); ok {
		return imp.ImportFrom(path, dir, mode)
	}
	return nil, fmt.Errorf("unsupported importer")
}

func (m moduleImporter) importGoat() (*types.Package, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, m.root, func(fi fs.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		return nil, err
	}
	pkg, ok := pkgs[goatPackageName]
	if !ok {
		return nil, fmt.Errorf("goat package not found")
	}
	files := make([]*ast.File, 0, len(pkg.Files))
	for _, f := range pkg.Files {
		files = append(files, f)
	}
	conf := types.Config{Importer: m.std, FakeImportC: true}
	return conf.Check(goatPackageFullPath, fset, files, nil)
}

func loadPackageWithTypes(packagePath string) (*PackageInfo, error) {
	fset := token.NewFileSet()
	parsed, err := parser.ParseDir(fset, packagePath, func(fi fs.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to parse package: %w", err)
	}
	if len(parsed) != 1 {
		return nil, fmt.Errorf("expected 1 package, got %d", len(parsed))
	}
	var astPkg *ast.Package
	for _, p := range parsed {
		astPkg = p
	}
	files := make([]*ast.File, 0, len(astPkg.Files))
	for _, f := range astPkg.Files {
		files = append(files, f)
	}
	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Implicits:  make(map[ast.Node]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
		Scopes:     make(map[ast.Node]*types.Scope),
	}
	abs, err := filepath.Abs(packagePath)
	if err != nil {
		return nil, err
	}
	conf := types.Config{Importer: newModuleImporter(findModuleRoot(abs)), FakeImportC: true}
	if _, err := conf.Check(astPkg.Name, fset, files, info); err != nil {
		return nil, fmt.Errorf("type checking failed: %w", err)
	}
	return &PackageInfo{Fset: fset, Syntax: files, TypesInfo: info}, nil
}

func findModuleRoot(path string) string {
	dir := path
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return path
		}
		dir = parent
	}
}
