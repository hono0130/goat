package mermaid

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/build"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	goatPackageName     = "goat"
	goatPackageFullPath = "github.com/goatx/goat"
)

type packageInfo struct {
	Fset      *token.FileSet
	Syntax    []*ast.File
	TypesInfo *types.Info
}

func loadPackageWithTypes(packagePath string) (packageInfo, error) {
	abs, err := filepath.Abs(packagePath)
	if err != nil {
		return packageInfo{}, err
	}

	moduleRoot, err := findModuleRoot(abs)
	if err != nil {
		return packageInfo{}, err
	}

	modulePath, err := readModulePath(moduleRoot)
	if err != nil {
		return packageInfo{}, err
	}

	importPath, err := packageImportPath(moduleRoot, modulePath, abs)
	if err != nil {
		return packageInfo{}, err
	}

	files, info, fset, err := parseAndTypeCheck(importPath, abs, moduleRoot, modulePath)
	if err != nil {
		return packageInfo{}, err
	}

	return packageInfo{Fset: fset, Syntax: files, TypesInfo: info}, nil
}

func parseAndTypeCheck(importPath, dir, moduleRoot, modulePath string) ([]*ast.File, *types.Info, *token.FileSet, error) {
	buildPkg, err := build.Default.ImportDir(dir, 0)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to inspect directory %s: %w", dir, err)
	}

	fset := token.NewFileSet()
	fileNames := append([]string{}, buildPkg.GoFiles...)
	fileNames = append(fileNames, buildPkg.CgoFiles...)
	files := make([]*ast.File, 0, len(fileNames))
	for _, name := range fileNames {
		filePath := filepath.Join(dir, name)
		file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to parse %s: %w", filePath, err)
		}
		files = append(files, file)
	}

	if len(files) == 0 {
		return nil, nil, nil, fmt.Errorf("no Go files found in %s", dir)
	}

	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Implicits:  make(map[ast.Node]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
		Scopes:     make(map[ast.Node]*types.Scope),
	}

	pkgImporter := newModuleImporter(moduleRoot, modulePath)
	conf := types.Config{
		Importer:    pkgImporter,
		FakeImportC: true,
		Sizes:       types.SizesFor("gc", build.Default.GOARCH),
	}

	if _, err := conf.Check(importPath, fset, files, info); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to type-check package %s: %w", importPath, err)
	}

	return files, info, fset, nil
}

func findModuleRoot(start string) (string, error) {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("failed to stat go.mod in %s: %w", dir, err)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found from %s", start)
		}
		dir = parent
	}
}

func readModulePath(moduleRoot string) (string, error) {
	goModPath := filepath.Join(moduleRoot, "go.mod")
	// #nosec G304 -- moduleRoot is discovered from the current module tree.
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return "", fmt.Errorf("failed to read go.mod: %w", err)
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				return fields[1], nil
			}
		}
	}
	return "", fmt.Errorf("module path not found in go.mod at %s", moduleRoot)
}

func packageImportPath(moduleRoot, modulePath, dir string) (string, error) {
	rel, err := filepath.Rel(moduleRoot, dir)
	if err != nil {
		return "", fmt.Errorf("failed to determine relative path: %w", err)
	}
	if rel == "." {
		return modulePath, nil
	}
	if strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("directory %s is outside module root %s", dir, moduleRoot)
	}
	return modulePath + "/" + filepath.ToSlash(rel), nil
}

type moduleImporter struct {
	moduleRoot string
	modulePath string
	fallback   types.Importer

	cache map[string]*types.Package
}

func newModuleImporter(moduleRoot, modulePath string) *moduleImporter {
	return &moduleImporter{
		moduleRoot: moduleRoot,
		modulePath: modulePath,
		fallback:   importer.Default(),
		cache:      make(map[string]*types.Package),
	}
}

func (m *moduleImporter) Import(path string) (*types.Package, error) {
	if pkg, ok := m.cache[path]; ok {
		return pkg, nil
	}

	if !strings.HasPrefix(path, m.modulePath) {
		if isStandardLibraryPackage(path) {
			return m.fallback.Import(path)
		}
		return m.importExternal(path)
	}

	rel := strings.TrimPrefix(path, m.modulePath)
	rel = strings.TrimPrefix(rel, "/")
	dir := filepath.Join(m.moduleRoot, filepath.FromSlash(rel))
	return m.importFromDir(path, dir)
}

func (m *moduleImporter) importFromDir(path, dir string) (*types.Package, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", dir)
	}

	buildPkg, err := build.Default.ImportDir(dir, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to load package in %s: %w", dir, err)
	}

	if pkg, ok := m.cache[path]; ok {
		return pkg, nil
	}
	placeholder := types.NewPackage(path, buildPkg.Name)
	m.cache[path] = placeholder

	fileNames := append([]string{}, buildPkg.GoFiles...)
	fileNames = append(fileNames, buildPkg.CgoFiles...)
	files, fset, err := parseGoFiles(dir, fileNames)
	if err != nil {
		delete(m.cache, path)
		return nil, fmt.Errorf("failed to load source files for %s: %w", path, err)
	}

	conf := types.Config{
		Importer:    m,
		FakeImportC: true,
		Sizes:       types.SizesFor("gc", build.Default.GOARCH),
	}

	pkg, err := conf.Check(path, fset, files, nil)
	if err != nil {
		delete(m.cache, path)
		return nil, fmt.Errorf("failed to type-check dependency %s: %w", path, err)
	}

	m.cache[path] = pkg

	return pkg, nil
}

func (m *moduleImporter) importExternal(path string) (*types.Package, error) {
	pkgInfo, err := goList(path, m.moduleRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to load external package %s: %w", path, err)
	}

	if pkg, ok := m.cache[path]; ok {
		return pkg, nil
	}
	placeholder := types.NewPackage(path, pkgInfo.Name)
	m.cache[path] = placeholder

	fileNames := append([]string{}, pkgInfo.GoFiles...)
	fileNames = append(fileNames, pkgInfo.CgoFiles...)
	files, fset, err := parseGoFiles(pkgInfo.Dir, fileNames)
	if err != nil {
		delete(m.cache, path)
		return nil, fmt.Errorf("failed to load source files for %s: %w", path, err)
	}

	conf := types.Config{
		Importer:    m,
		FakeImportC: true,
		Sizes:       types.SizesFor("gc", build.Default.GOARCH),
	}

	pkg, err := conf.Check(path, fset, files, nil)
	if err != nil {
		delete(m.cache, path)
		return nil, fmt.Errorf("failed to type-check dependency %s: %w", path, err)
	}

	m.cache[path] = pkg

	return pkg, nil
}

func parseGoFiles(dir string, names []string) ([]*ast.File, *token.FileSet, error) {
	fset := token.NewFileSet()
	files := make([]*ast.File, 0, len(names))
	for _, name := range names {
		filePath := filepath.Join(dir, name)
		file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse %s: %w", filePath, err)
		}
		files = append(files, file)
	}

	if len(files) == 0 {
		return nil, nil, fmt.Errorf("no Go files for package in %s", dir)
	}

	return files, fset, nil
}

func isStandardLibraryPackage(path string) bool {
	return !strings.Contains(path, ".")
}

type goListPackage struct {
	Name     string
	Dir      string
	GoFiles  []string
	CgoFiles []string
	Error    *struct {
		Err string
	}
}

func goList(path, moduleRoot string) (*goListPackage, error) {
	cmd := exec.Command("go", "list", "-json", path)
	cmd.Dir = moduleRoot
	cmd.Env = os.Environ()
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("go list %s failed: %s", path, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("go list %s failed: %w", path, err)
	}

	var pkg goListPackage
	if err := json.Unmarshal(out, &pkg); err != nil {
		return nil, fmt.Errorf("failed to decode go list output: %w", err)
	}

	if pkg.Error != nil && pkg.Error.Err != "" {
		return nil, errors.New(pkg.Error.Err)
	}
	if pkg.Dir == "" {
		return nil, fmt.Errorf("go list returned empty directory for %s", path)
	}
	if pkg.Name == "" {
		return nil, fmt.Errorf("go list returned empty package name for %s", path)
	}

	return &pkg, nil
}
