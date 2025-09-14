package mermaid

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"sort"
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

func stateMachineOrder(pkg *PackageInfo) []string {
	var order []string
	seenStateMachines := make(map[string]bool)

	for _, file := range pkg.Syntax {
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
			if len(callExpr.Args) < 3 {
				return true
			}
			var stateMachineType string
			if tv, ok := pkg.TypesInfo.Types[callExpr.Args[0]]; ok && tv.Type != nil {
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
			handlerFunc, ok := callExpr.Args[len(callExpr.Args)-1].(*ast.FuncLit)
			if !ok {
				return true
			}
			var eventType string
			if handlerType == onEventHandler && len(callExpr.Args) >= 4 {
				eventType = getEventType(callExpr.Args[2], pkg)
			}
			pos := pkg.Fset.Position(handlerFunc.Pos())
			handlerID := fmt.Sprintf("%s_%s_%s_%s:%d",
				stateMachineType,
				handlerType,
				eventType,
				filepath.Base(pos.Filename),
				pos.Line)
			ast.Inspect(handlerFunc.Body, func(n ast.Node) bool {
				sendToCall, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
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

func isFromGoat(sel *ast.SelectorExpr, info *types.Info) bool {
	id, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	if obj := info.Uses[id]; obj != nil {
		if pkgName, ok := obj.(*types.PkgName); ok {
			imported := pkgName.Imported()
			if imported != nil {
				p := imported.Path()
				return p == goatPackageFullPath
			}
		}
		if obj.Pkg() != nil {
			p := obj.Pkg().Path()
			return p == goatPackageFullPath
		}
	}
	return id.Name == goatPackageName
}

func getTypeName(expr ast.Expr, pkg *PackageInfo, isEvent bool) string {
	if tv, ok := pkg.TypesInfo.Types[expr]; ok && tv.Type != nil {
		typ := tv.Type
		if ptr, ok := typ.(*types.Pointer); ok {
			typ = ptr.Elem()
		}
		if named, ok := typ.(*types.Named); ok {
			return named.Obj().Name()
		}
	}
	if isEvent {
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
	}
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return e.Sel.Name
	}
	return unknownType
}

func resolveTargetType(targetExpr ast.Expr, pkg *PackageInfo) string {
	return getTypeName(targetExpr, pkg, false)
}

func getEventType(eventExpr ast.Expr, pkg *PackageInfo) string {
	return getTypeName(eventExpr, pkg, true)
}
