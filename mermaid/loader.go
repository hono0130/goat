package mermaid

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
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

func loadPackageWithTypes(packagePath string) (*packageInfo, error) {
	abs, err := filepath.Abs(packagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve package path %s: %w", packagePath, err)
	}

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo |
			packages.NeedModule | packages.NeedDeps,
		Dir: abs,
	}

	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		return nil, fmt.Errorf("failed to load package in %s: %w", abs, err)
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no packages found in %s", abs)
	}

	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		var b strings.Builder
		for i, pkgErr := range pkg.Errors {
			if i > 0 {
				b.WriteString("\n")
			}
			b.WriteString(pkgErr.Error())
		}
		return nil, fmt.Errorf("failed to load package in %s: %s", abs, b.String())
	}
	if pkg.TypesInfo == nil {
		return nil, fmt.Errorf("failed to obtain type information for package in %s", abs)
	}

	info := &packageInfo{
		Fset:      pkg.Fset,
		Syntax:    pkg.Syntax,
		TypesInfo: pkg.TypesInfo,
	}
	return info, nil
}

func isStandardLibraryPackage(path string) bool {
	return !strings.Contains(path, ".")
}
