package mermaid

import (
	"fmt"
	"go/ast"
	"go/types"
	"path/filepath"
	"sort"

	"github.com/goatx/goat-cli/internal/load"
)

const (
	stateMachineType = "StateMachine"
	sendToFunction   = "SendTo"
	onEntryHandler   = "OnEntry"
	onEventHandler   = "OnEvent"
	onExitHandler    = "OnExit"
)

type sequenceDiagram struct {
	participants []string
	elements     []element
}

type element struct {
	flow     flow
	branches []branch
}

type branch struct {
	flow     flow
	elements []element
}

type flow struct {
	from             string
	to               string
	eventType        string
	handlerType      string
	handlerEventType string
	handlerID        string
	fileName         string
	line             int
}

func analyze(pkg *load.PackageInfo) (*sequenceDiagram, error) {
	order, err := stateMachineOrder(pkg)
	if err != nil {
		return nil, err
	}
	flows, err := communicationFlows(pkg)
	if err != nil {
		return nil, err
	}
	elements := buildElements(flows)

	return &sequenceDiagram{
		participants: order,
		elements:     elements,
	}, nil
}

func stateMachineOrder(pkg *load.PackageInfo) ([]string, error) {
	order := make([]string, 0)
	seenStateMachines := make(map[string]bool)

	for _, file := range pkg.Syntax {
		var inspectErr error
		ast.Inspect(file, func(n ast.Node) bool {
			typeSpec, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				return true
			}

			for _, field := range structType.Fields.List {
				if field.Names != nil {
					continue
				}
				selExpr, ok := field.Type.(*ast.SelectorExpr)
				if !ok {
					continue
				}
				fromGoat, err := isFromGoat(selExpr, pkg.TypesInfo)
				if err != nil {
					inspectErr = err
					return false
				}
				if !fromGoat {
					continue
				}
				if selExpr.Sel.Name != stateMachineType {
					continue
				}
				name := typeSpec.Name.Name
				if !seenStateMachines[name] {
					order = append(order, name)
					seenStateMachines[name] = true
				}
				break
			}
			return true
		})
		if inspectErr != nil {
			return nil, inspectErr
		}
	}
	return order, nil
}

func communicationFlows(pkg *load.PackageInfo) ([]flow, error) {
	flows := make([]flow, 0)
	var inspectErr error

	for _, file := range pkg.Syntax {
		if inspectErr != nil {
			break
		}
		ast.Inspect(file, func(n ast.Node) bool {
			if inspectErr != nil {
				return false
			}

			callExpr, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			handler, found, err := extractHandlerInfo(callExpr, pkg)
			if err != nil {
				inspectErr = err
				return false
			}
			if !found {
				return true
			}

			ast.Inspect(handler.function.Body, func(node ast.Node) bool {
				if inspectErr != nil {
					return false
				}

				sendCall, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}

				f, ok, err := extractSendToFlow(sendCall, pkg, handler)
				if err != nil {
					inspectErr = err
					return false
				}
				if ok {
					flows = append(flows, f)
				}

				return true
			})

			return inspectErr == nil
		})
	}

	if inspectErr != nil {
		return nil, inspectErr
	}

	seen := make(map[string]bool)
	unique := make([]flow, 0)
	for _, f := range flows {
		key := fmt.Sprintf("%s->%s:%s:%s:%s", f.from, f.to, f.eventType, f.handlerType, f.handlerEventType)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, f)
		}
	}

	sort.SliceStable(unique, func(i, j int) bool {
		iFlow := unique[i]
		jFlow := unique[j]
		if iFlow.fileName != jFlow.fileName {
			return iFlow.fileName < jFlow.fileName
		}
		return iFlow.line < jFlow.line

	})

	return unique, nil
}

type handlerInfo struct {
	kind         string
	stateMachine string
	handlerEvent string
	handlerID    string
	function     *ast.FuncLit
}

func extractHandlerInfo(callExpr *ast.CallExpr, pkg *load.PackageInfo) (*handlerInfo, bool, error) {
	selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil, false, nil
	}
	fromGoat, err := isFromGoat(selExpr, pkg.TypesInfo)
	if err != nil {
		return nil, false, err
	}
	if !fromGoat {
		return nil, false, nil
	}

	var kind string
	switch selExpr.Sel.Name {
	case onEntryHandler, onEventHandler, onExitHandler:
		kind = selExpr.Sel.Name
	default:
		return nil, false, nil
	}

	if len(callExpr.Args) < 3 {
		return nil, false, nil
	}

	stateMachine := resolveStateMachineType(callExpr.Args[0], pkg.TypesInfo)
	if stateMachine == "" {
		return nil, false, nil
	}

	handlerFunc, ok := callExpr.Args[len(callExpr.Args)-1].(*ast.FuncLit)
	if !ok {
		return nil, false, nil
	}

	eventType := ""
	if kind == onEventHandler && len(callExpr.Args) >= 4 {
		if name, ok := namedTypeName(callExpr.Args[2], pkg.TypesInfo); ok {
			eventType = name
		}
	}
	handlerID := buildHandlerID(stateMachine, kind, eventType, handlerFunc, pkg)

	return &handlerInfo{
		kind:         kind,
		stateMachine: stateMachine,
		handlerEvent: eventType,
		handlerID:    handlerID,
		function:     handlerFunc,
	}, true, nil
}

func resolveStateMachineType(expr ast.Expr, info *types.Info) string {
	tv, ok := info.Types[expr]
	if !ok || tv.Type == nil {
		return ""
	}
	typ := tv.Type
	if ptr, ok := typ.(*types.Pointer); ok {
		typ = ptr.Elem()
	}
	named, ok := typ.(*types.Named)
	if !ok {
		return ""
	}
	typeArgs := named.TypeArgs()
	if typeArgs == nil || typeArgs.Len() == 0 {
		return ""
	}
	arg := typeArgs.At(0)
	if p, ok := arg.(*types.Pointer); ok {
		arg = p.Elem()
	}
	if n, ok := arg.(*types.Named); ok {
		return n.Obj().Name()
	}
	return ""
}

func buildHandlerID(stateMachine, kind, eventType string, handlerFunc *ast.FuncLit, pkg *load.PackageInfo) string {
	pos := pkg.Fset.Position(handlerFunc.Pos())
	return fmt.Sprintf("%s_%s_%s_%s:%d",
		stateMachine,
		kind,
		eventType,
		filepath.Base(pos.Filename),
		pos.Line)
}

func extractSendToFlow(callExpr *ast.CallExpr, pkg *load.PackageInfo, info *handlerInfo) (flow, bool, error) {
	selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok {
		return flow{}, false, nil
	}
	if selExpr.Sel.Name != sendToFunction {
		return flow{}, false, nil
	}

	fromGoat, err := isFromGoat(selExpr, pkg.TypesInfo)
	if err != nil {
		return flow{}, false, err
	}
	if !fromGoat {
		return flow{}, false, nil
	}

	if len(callExpr.Args) != 3 {
		return flow{}, false, fmt.Errorf("send to flow call expression has %d arguments, expected 3", len(callExpr.Args))
	}

	toName := "UnknownStatemachine"
	if name, ok := namedTypeName(callExpr.Args[1], pkg.TypesInfo); ok && name != "AbstractStateMachine" {
		toName = name
	}

	var evName string
	if name, ok := namedTypeName(callExpr.Args[2], pkg.TypesInfo); ok {
		evName = name
	}

	pos := pkg.Fset.Position(callExpr.Pos())

	return flow{
		from:             info.stateMachine,
		to:               toName,
		eventType:        evName,
		handlerType:      info.kind,
		handlerEventType: info.handlerEvent,
		handlerID:        info.handlerID,
		fileName:         filepath.Base(pos.Filename),
		line:             pos.Line,
	}, true, nil
}

func buildElements(flows []flow) []element {
	handlerGroups := make(map[string][]flow)
	for _, f := range flows {
		handlerGroups[f.handlerID] = append(handlerGroups[f.handlerID], f)
	}

	emitted := make(map[string]bool)
	result := make([]element, 0)

	var build func(handlerID string, path map[string]bool) []element

	build = func(handlerID string, path map[string]bool) []element {
		if path[handlerID] {
			return nil
		}
		if emitted[handlerID] {
			return nil
		}

		handlerFlows := handlerGroups[handlerID]
		if len(handlerFlows) == 0 {
			emitted[handlerID] = true
			return nil
		}

		emitted[handlerID] = true

		nextPath := copyVisited(path)
		nextPath[handlerID] = true

		if len(handlerFlows) == 1 {
			current := handlerFlows[0]
			elem := element{flow: current}
			nextHandlers := uniqueNextHandlers(current, flows)
			elements := make([]element, 0, len(nextHandlers)+1)
			elements = append(elements, elem)
			for _, nextID := range nextHandlers {
				child := build(nextID, nextPath)
				elements = append(elements, child...)
			}
			return elements
		}

		sortedFlows := make([]flow, 0, len(handlerFlows))
		sortedFlows = append(sortedFlows, handlerFlows...)
		sort.Slice(sortedFlows, func(i, j int) bool {
			fi := sortedFlows[i]
			fj := sortedFlows[j]

			ni := len(uniqueNextHandlers(fi, flows))
			nj := len(uniqueNextHandlers(fj, flows))
			if ni != nj {
				return ni < nj
			}
			if fi.fileName != fj.fileName {
				return fi.fileName < fj.fileName
			}
			return fi.line < fj.line
		})

		branches := make([]branch, 0, len(sortedFlows))
		for _, hf := range sortedFlows {
			br := branch{
				flow: hf,
			}
			nextHandlers := uniqueNextHandlers(hf, flows)
			for _, nextID := range nextHandlers {
				child := build(nextID, nextPath)
				br.elements = append(br.elements, child...)
			}
			branches = append(branches, br)
		}

		return []element{{branches: branches}}
	}

	for _, f := range flows {
		if f.handlerType == onEntryHandler {
			elements := build(f.handlerID, map[string]bool{})
			result = append(result, elements...)
		}
	}

	for _, f := range flows {
		if !emitted[f.handlerID] {
			elements := build(f.handlerID, map[string]bool{})
			result = append(result, elements...)
		}
	}

	return result
}

func copyVisited(path map[string]bool) map[string]bool {
	if len(path) == 0 {
		return make(map[string]bool)
	}
	copied := make(map[string]bool, len(path))
	for k, v := range path {
		copied[k] = v
	}
	return copied
}

func uniqueNextHandlers(current flow, flows []flow) []string {
	result := make([]string, 0)
	seen := make(map[string]bool)
	for _, candidate := range flows {
		if candidate.handlerType == onEventHandler &&
			candidate.handlerEventType == current.eventType &&
			candidate.from == current.to &&
			!seen[candidate.handlerID] {
			seen[candidate.handlerID] = true
			result = append(result, candidate.handlerID)
		}
	}
	return result
}

func isFromGoat(sel *ast.SelectorExpr, info *types.Info) (bool, error) {
	id, ok := sel.X.(*ast.Ident)
	if !ok {
		return false, nil
	}
	obj, ok := info.Uses[id]
	if !ok {
		return false, fmt.Errorf("missing type info for identifier %q", id.Name)
	}
	if pkgName, ok := obj.(*types.PkgName); ok {
		if imported := pkgName.Imported(); imported != nil {
			return imported.Path() == load.GoatPackageFullPath, nil
		}
		return false, fmt.Errorf("unexpected nil imported package for %q", id.Name)
	}
	if pkg := obj.Pkg(); pkg != nil {
		return pkg.Path() == load.GoatPackageFullPath, nil
	}
	return false, fmt.Errorf("object for identifier %q has no package: %T", id.Name, obj)
}

func namedTypeName(expr ast.Expr, info *types.Info) (string, bool) {
	if tv, ok := info.Types[expr]; ok && tv.Type != nil {
		typ := tv.Type
		if ptr, ok := typ.(*types.Pointer); ok {
			typ = ptr.Elem()
		}
		if named, ok := typ.(*types.Named); ok {
			return named.Obj().Name(), true
		}
	}
	return "", false
}
