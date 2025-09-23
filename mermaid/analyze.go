package mermaid

import (
	"fmt"
	"go/ast"
	"go/types"
	"path/filepath"
	"sort"
)

const (
	stateMachineType = "StateMachine"
	sendToFunction   = "SendTo"
	unknownType      = "Unknown"
	onEntryHandler   = "OnEntry"
	onEventHandler   = "OnEvent"
	onExitHandler    = "OnExit"
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
	flows    []flow
	branches [][]flow
}

func stateMachineOrder(pkg *packageInfo) []string {
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

func communicationFlows(pkg *packageInfo) []flow {
	var flows []flow
	for _, file := range pkg.Syntax {
		ast.Inspect(file, func(n ast.Node) bool {
			callExpr, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			ctx, ok := extractHandlerContext(callExpr, pkg)
			if !ok {
				return true
			}
			ast.Inspect(ctx.function.Body, func(node ast.Node) bool {
				sendCall, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				if f, ok := extractSendToFlow(sendCall, pkg, ctx); ok {
					flows = append(flows, f)
				}
				return true
			})
			return true
		})
	}
	return uniqueFlows(flows)
}

type handlerContext struct {
	kind         string
	stateMachine string
	handlerEvent string
	handlerID    string
	function     *ast.FuncLit
}

func extractHandlerContext(callExpr *ast.CallExpr, pkg *packageInfo) (*handlerContext, bool) {
	selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok || !isFromGoat(selExpr, pkg.TypesInfo) {
		return nil, false
	}

	var kind string
	switch selExpr.Sel.Name {
	case onEntryHandler, onEventHandler, onExitHandler:
		kind = selExpr.Sel.Name
	default:
		return nil, false
	}
	if len(callExpr.Args) < 3 {
		return nil, false
	}

	stateMachine := resolveStateMachineType(callExpr.Args[0], pkg.TypesInfo)
	if stateMachine == "" {
		return nil, false
	}

	handlerFunc, ok := callExpr.Args[len(callExpr.Args)-1].(*ast.FuncLit)
	if !ok {
		return nil, false
	}

	eventType := ""
	if kind == onEventHandler && len(callExpr.Args) >= 4 {
		if name, ok := namedTypeName(callExpr.Args[2], pkg.TypesInfo); ok {
			eventType = name
		}
	}
	handlerID := buildHandlerID(stateMachine, kind, eventType, handlerFunc, pkg)

	return &handlerContext{
		kind:         kind,
		stateMachine: stateMachine,
		handlerEvent: eventType,
		handlerID:    handlerID,
		function:     handlerFunc,
	}, true
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

func buildHandlerID(stateMachine, kind, eventType string, handlerFunc *ast.FuncLit, pkg *packageInfo) string {
	pos := pkg.Fset.Position(handlerFunc.Pos())
	return fmt.Sprintf("%s_%s_%s_%s:%d",
		stateMachine,
		kind,
		eventType,
		filepath.Base(pos.Filename),
		pos.Line)
}

func extractSendToFlow(callExpr *ast.CallExpr, pkg *packageInfo, ctx *handlerContext) (flow, bool) {
	selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok || !isFromGoat(selExpr, pkg.TypesInfo) || selExpr.Sel.Name != sendToFunction {
		return flow{}, false
	}
	if len(callExpr.Args) < 3 {
		return flow{}, false
	}
	toName := unknownType
	if name, ok := namedTypeName(callExpr.Args[1], pkg.TypesInfo); ok {
		toName = name
	}
	evName := ""
	if name, ok := namedTypeName(callExpr.Args[2], pkg.TypesInfo); ok {
		evName = name
	}
	return flow{
		from:             ctx.stateMachine,
		to:               toName,
		eventType:        evName,
		handlerType:      ctx.kind,
		handlerEventType: ctx.handlerEvent,
		handlerID:        ctx.handlerID,
	}, true
}

func uniqueFlows(flows []flow) []flow {
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

func findNextFlows(f *flow, flows []flow) []flow {
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

func findFlowPosition(target *flow, flows []flow) int {
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

func collectChain(f *flow, flows []flow, processed map[string]bool) []flow {
	var chain []flow
	for _, next := range findNextFlows(f, flows) {
		if processed[next.handlerID] {
			continue
		}
		processed[next.handlerID] = true
		chain = append(chain, next)
		chain = append(chain, collectChain(&next, flows, processed)...)
	}
	return chain
}

type elementBuilder struct {
	flows         []flow
	processed     map[string]bool
	handlerGroups map[string][]flow
	elements      []element
}

func (b *elementBuilder) process(f *flow) {
	if b.processed[f.handlerID] {
		return
	}
	handlerFlows := b.handlerGroups[f.handlerID]
	if len(handlerFlows) > 1 {
		var trigger flow
		triggerFound := false
		for _, cand := range b.flows {
			if cand.eventType == f.handlerEventType && cand.to == f.from {
				trigger = cand
				triggerFound = true
				break
			}
		}
		if triggerFound && !b.processed[trigger.handlerID] {
			// Add the triggering flow as a normal element before the branches
			b.elements = append(b.elements, element{flows: []flow{trigger}})
			b.processed[trigger.handlerID] = true
		}
		sorted := append([]flow(nil), handlerFlows...)
		sort.Slice(sorted, func(i, j int) bool {
			fi, fj := sorted[i], sorted[j]
			ci := len(findNextFlows(&fi, b.flows))
			cj := len(findNextFlows(&fj, b.flows))
			if ci != cj {
				return ci < cj
			}
			return findFlowPosition(&fi, b.flows) < findFlowPosition(&fj, b.flows)
		})
		var branchPaths [][]flow
		for _, h := range sorted {
			path := append([]flow{h}, collectChain(&h, b.flows, b.processed)...)
			branchPaths = append(branchPaths, path)
		}
		b.elements = append(b.elements, element{branches: branchPaths})
		b.processed[f.handlerID] = true
		return
	}
	h := handlerFlows[0]
	b.elements = append(b.elements, element{flows: []flow{h}})
	b.processed[h.handlerID] = true
	for _, next := range findNextFlows(&h, b.flows) {
		nextCopy := next
		b.process(&nextCopy)
	}
}

func buildElements(flows []flow) []element {
	b := &elementBuilder{
		flows:         flows,
		processed:     make(map[string]bool),
		handlerGroups: make(map[string][]flow),
	}
	for i := range flows {
		f := flows[i]
		b.handlerGroups[f.handlerID] = append(b.handlerGroups[f.handlerID], f)
	}
	for i := range flows {
		f := &flows[i]
		if f.handlerType == onEntryHandler && !b.processed[f.handlerID] {
			b.process(f)
		}
	}
	for i := range flows {
		f := &flows[i]
		if !b.processed[f.handlerID] {
			b.process(f)
		}
	}
	return b.elements
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
