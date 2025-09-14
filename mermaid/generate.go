package mermaid

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	goatPackageName     = "goat"
	goatPackageFullPath = "github.com/goatx/goat"
	stateMachineType    = "StateMachine"
	sendToFunction      = "SendTo"
	unknownType         = "Unknown"

	onEntryHandler = "OnEntry"
	onEventHandler = "OnEvent"
	onExitHandler  = "OnExit"
)

type flow struct {
	from             string
	to               string
	eventType        string
	handlerType      string
	handlerEventType string
	handlerID        string
}

type element struct {
	flows      []flow
	isOptional bool
}

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

// Generate analyzes a Go package and writes a Mermaid sequence diagram to writer.
func Generate(packagePath string, writer io.Writer) error {
	pkg, err := loadPackageWithTypes(packagePath)
	if err != nil {
		return fmt.Errorf("failed to load package with types: %w", err)
	}
	order := stateMachineOrder(pkg)
	flows := communicationFlows(pkg)
	elements := buildElements(flows)
	_, err = writer.Write([]byte(render(elements, order)))
	return err
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

func stateMachineOrder(pkg *PackageInfo) []string {
	var order []string
	seenStateMachines := make(map[string]bool)

	for _, file := range pkg.Syntax {
		ast.Inspect(file, func(n ast.Node) bool {
			typeSpec, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}

			// Check if this is a state machine struct (embeds goat.StateMachine)
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				return true
			}

			for _, field := range structType.Fields.List {
				if field.Names != nil { // not an embedded field
					continue
				}

				selExpr, ok := field.Type.(*ast.SelectorExpr)
				if !ok {
					continue
				}

				if isFromGoat(selExpr, pkg.TypesInfo) && selExpr.Sel.Name == stateMachineType {
					name := typeSpec.Name.Name
					if !seenStateMachines[name] {
						order = append(order, name)
						seenStateMachines[name] = true
					}
					break
				}
			}
			return true
		})
	}

	return order
}

func communicationFlows(pkg *PackageInfo) []flow {
	var flows []flow

	for _, file := range pkg.Syntax {
		ast.Inspect(file, func(n ast.Node) bool {
			callExpr, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			// Check if this is a goat handler registration (OnEntry, OnEvent, OnExit)
			selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			if !isFromGoat(selExpr, pkg.TypesInfo) {
				return true
			}

			var handlerType string
			switch selExpr.Sel.Name {
			case onEntryHandler, onEventHandler, onExitHandler:
				handlerType = selExpr.Sel.Name
			default:
				return true
			}

			// Extract state machine type from spec parameter
			if len(callExpr.Args) < 3 {
				return true
			}

			var stateMachineType string
			if tv, ok := pkg.TypesInfo.Types[callExpr.Args[0]]; ok && tv.Type != nil {
				// Extract state machine type from generic spec type
				typ := tv.Type
				if ptr, ok := typ.(*types.Pointer); ok {
					typ = ptr.Elem()
				}
				if named, ok := typ.(*types.Named); ok {
					if typeArgs := named.TypeArgs(); typeArgs != nil && typeArgs.Len() > 0 {
						arg := typeArgs.At(0)
						if p, ok := arg.(*types.Pointer); ok {
							arg = p.Elem()
						}
						if n, ok := arg.(*types.Named); ok {
							stateMachineType = n.Obj().Name()
						}
					}
				}
			}
			if stateMachineType == "" {
				return true
			}

			// Extract handler function
			handlerFunc, ok := callExpr.Args[len(callExpr.Args)-1].(*ast.FuncLit)
			if !ok {
				return true
			}

			// Extract event type for OnEvent handlers
			var eventType string
			if handlerType == onEventHandler && len(callExpr.Args) >= 4 {
				eventType = getEventType(callExpr.Args[2], pkg)
			}

			// Generate unique ID for handler based on source position
			pos := pkg.Fset.Position(handlerFunc.Pos())
			handlerID := fmt.Sprintf("%s_%s_%s_%s:%d",
				stateMachineType,
				handlerType,
				eventType,
				filepath.Base(pos.Filename),
				pos.Line)

			// Find all goat.SendTo calls in the handler function
			ast.Inspect(handlerFunc.Body, func(n ast.Node) bool {
				sendToCall, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}

				// Check if this is a goat.SendTo call
				selExpr, ok := sendToCall.Fun.(*ast.SelectorExpr)
				if !ok || !isFromGoat(selExpr, pkg.TypesInfo) || selExpr.Sel.Name != sendToFunction {
					return true
				}

				if len(sendToCall.Args) < 3 {
					return true
				}

				f := flow{
					from:             stateMachineType,
					to:               resolveTargetType(sendToCall.Args[1], pkg),
					eventType:        getEventType(sendToCall.Args[2], pkg),
					handlerType:      handlerType,
					handlerEventType: eventType,
					handlerID:        handlerID,
				}
				flows = append(flows, f)

				return true
			})

			return true
		})
	}

	// Remove duplicates
	seen := make(map[string]bool)
	var unique []flow
	for _, f := range flows {
		key := fmt.Sprintf("%s->%s:%s:%s:%s", f.from, f.to, f.eventType, f.handlerType, f.handlerEventType)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, f)
		}
	}
	return unique
}

func isFromGoat(sel *ast.SelectorExpr, info *types.Info) bool {
	id, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	if obj := info.Uses[id]; obj != nil {
		// Handle *types.PkgName objects (package references)
		if pkgName, ok := obj.(*types.PkgName); ok {
			imported := pkgName.Imported()
			if imported != nil {
				p := imported.Path()
				return p == goatPackageFullPath
			}
		}
		// Handle other object types
		if obj.Pkg() != nil {
			p := obj.Pkg().Path()
			return p == goatPackageFullPath
		}
	}
	// Fallback to identifier name check for backward compatibility
	return id.Name == goatPackageName
}

func getTypeName(expr ast.Expr, pkg *PackageInfo, isEvent bool) string {
	// Try to get type from type information first
	if tv, ok := pkg.TypesInfo.Types[expr]; ok && tv.Type != nil {
		typ := tv.Type
		if ptr, ok := typ.(*types.Pointer); ok {
			typ = ptr.Elem()
		}
		if named, ok := typ.(*types.Named); ok {
			return named.Obj().Name()
		}
	}

	// Fallback to AST parsing - different logic for events vs targets
	if isEvent {
		// Event type extraction from AST
		switch e := expr.(type) {
		case *ast.UnaryExpr:
			if e.Op == token.AND {
				return getTypeName(e.X, pkg, true)
			}
		case *ast.CompositeLit:
			switch t := e.Type.(type) {
			case *ast.Ident:
				return t.Name
			case *ast.SelectorExpr:
				return t.Sel.Name
			}
		}
		return ""
	} else {
		// Target type extraction from AST
		switch e := expr.(type) {
		case *ast.Ident:
			return e.Name
		case *ast.SelectorExpr:
			return e.Sel.Name
		}
		return unknownType
	}
}

func resolveTargetType(targetExpr ast.Expr, pkg *PackageInfo) string {
	return getTypeName(targetExpr, pkg, false)
}

func getEventType(eventExpr ast.Expr, pkg *PackageInfo) string {
	return getTypeName(eventExpr, pkg, true)
}

func groupFlowsByHandler(flows []flow) map[string][]flow {
	groups := make(map[string][]flow)
	for _, f := range flows {
		groups[f.handlerID] = append(groups[f.handlerID], f)
	}
	return groups
}

func findTriggerFlow(handlerFlow flow, flows []flow) *flow {
	for _, f := range flows {
		if f.eventType == handlerFlow.handlerEventType && f.to == handlerFlow.from {
			return &f
		}
	}
	return nil
}

func findNextFlows(f flow, flows []flow) []flow {
	var next []flow
	for _, candidate := range flows {
		if candidate.handlerType == onEventHandler &&
			candidate.handlerEventType == f.eventType &&
			candidate.from == f.to {
			next = append(next, candidate)
		}
	}
	return next
}

func findFlowPosition(target flow, flows []flow) int {
	for i, f := range flows {
		if f.handlerID == target.handlerID &&
			f.from == target.from &&
			f.to == target.to &&
			f.eventType == target.eventType {
			return i
		}
	}
	return len(flows)
}

func collectChain(f flow, flows []flow, processed map[string]bool) []flow {
	var chain []flow
	for _, next := range findNextFlows(f, flows) {
		if processed[next.handlerID] {
			continue
		}
		processed[next.handlerID] = true
		chain = append(chain, next)
		chain = append(chain, collectChain(next, flows, processed)...)
	}
	return chain
}

func buildElements(flows []flow) []element {
	var elements []element
	processed := make(map[string]bool)
	handlerGroups := groupFlowsByHandler(flows)

	var process func(flow)
	process = func(f flow) {
		if processed[f.handlerID] {
			return
		}

		handlerFlows := handlerGroups[f.handlerID]
		if len(handlerFlows) > 1 {
			if trigger := findTriggerFlow(f, flows); trigger != nil && !processed[trigger.handlerID] {
				elements = append(elements, element{flows: []flow{*trigger}, isOptional: true})
				processed[trigger.handlerID] = true
			}

			sorted := append([]flow(nil), handlerFlows...)
			sort.Slice(sorted, func(i, j int) bool {
				fi, fj := sorted[i], sorted[j]
				hasChainI := len(findNextFlows(fi, flows)) > 0
				hasChainJ := len(findNextFlows(fj, flows)) > 0
				if hasChainI != hasChainJ {
					return !hasChainI
				}
				return findFlowPosition(fi, flows) < findFlowPosition(fj, flows)
			})

			for _, h := range sorted {
				path := append([]flow{h}, collectChain(h, flows, processed)...)
				elements = append(elements, element{flows: path, isOptional: true})
			}
			processed[f.handlerID] = true
			return
		}

		h := handlerFlows[0]
		elements = append(elements, element{flows: []flow{h}})
		processed[h.handlerID] = true
		for _, next := range findNextFlows(h, flows) {
			process(next)
		}
	}

	for _, f := range flows {
		if f.handlerType == onEntryHandler && !processed[f.handlerID] {
			process(f)
		}
	}
	for _, f := range flows {
		if !processed[f.handlerID] {
			process(f)
		}
	}
	return elements
}

func render(elements []element, order []string) string {
	participants := orderedParticipants(elements, order)
	var sb strings.Builder
	sb.WriteString("sequenceDiagram\n")
	for _, p := range participants {
		sb.WriteString(fmt.Sprintf("    participant %s\n", p))
	}
	sb.WriteString("\n")
	for _, e := range elements {
		if e.isOptional {
			sb.WriteString("    opt\n")
			for _, f := range e.flows {
				sb.WriteString(fmt.Sprintf("        %s->>%s: %s\n", f.from, f.to, f.eventType))
			}
			sb.WriteString("    end\n")
			continue
		}
		for _, f := range e.flows {
			sb.WriteString(fmt.Sprintf("    %s->>%s: %s\n", f.from, f.to, f.eventType))
		}
	}
	return sb.String()
}

func orderedParticipants(elements []element, order []string) []string {
	seen := make(map[string]bool)
	var participants []string
	for _, p := range order {
		if !seen[p] {
			participants = append(participants, p)
			seen[p] = true
		}
	}
	var extras []string
	for _, e := range elements {
		for _, f := range e.flows {
			if !seen[f.from] {
				extras = append(extras, f.from)
				seen[f.from] = true
			}
			if !seen[f.to] {
				extras = append(extras, f.to)
				seen[f.to] = true
			}
		}
	}
	sort.Strings(extras)
	participants = append(participants, extras...)
	return participants
}
