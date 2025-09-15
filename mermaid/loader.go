package mermaid

import (
    "fmt"
    "go/ast"
    "go/token"
    "go/types"
    "path/filepath"

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
        return nil, err
    }
    cfg := &packages.Config{
        Mode:  packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo,
        Dir:   abs,
        Tests: false,
    }
    pkgs, err := packages.Load(cfg, "./")
    if err != nil {
        return nil, fmt.Errorf("failed to load package: %w", err)
    }
    if len(pkgs) == 0 {
        return nil, fmt.Errorf("no packages found at %s", abs)
    }
    if len(pkgs) != 1 {
        return nil, fmt.Errorf("expected exactly 1 package at %s, got %d; please provide a specific package directory", abs, len(pkgs))
    }
    p := pkgs[0]
    if len(p.Errors) > 0 {
        return nil, fmt.Errorf("failed to load package: %v", p.Errors[0])
    }
    return &packageInfo{Fset: p.Fset, Syntax: p.Syntax, TypesInfo: p.TypesInfo}, nil
}
