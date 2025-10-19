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

// Diagram captures the intermediate representation of a Mermaid sequence diagram.
type Diagram struct {
	Participants []string
	Elements     []Element
}

// Element models either a simple flow or an alternative branch in the diagram.
type Element struct {
	Flows    []Flow
	Branches [][]Flow
}

// Flow represents a directed edge between two state machines triggered by an event.
type Flow struct {
	From             string
	To               string
	EventType        string
	HandlerType      string
	HandlerEventType string
	HandlerID        string
}

// Analyze inspects the given package path and returns the diagram representation.
func Analyze(packagePath string) (*Diagram, error) {
	pkg, err := loadPackageWithTypes(packagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load package with types: %w", err)
	}

	order := stateMachineOrder(pkg)
	flows := communicationFlows(pkg)
	elements := buildElements(flows)

	return &Diagram{
		Participants: orderedParticipants(elements, order),
		Elements:     elements,
	}, nil
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

func communicationFlows(pkg *packageInfo) []Flow {
	var flows []Flow
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

func extractSendToFlow(callExpr *ast.CallExpr, pkg *packageInfo, ctx *handlerContext) (Flow, bool) {
	selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok || !isFromGoat(selExpr, pkg.TypesInfo) || selExpr.Sel.Name != sendToFunction {
		return Flow{}, false
	}
	if len(callExpr.Args) < 3 {
		return Flow{}, false
	}
	toName := unknownType
	if name, ok := namedTypeName(callExpr.Args[1], pkg.TypesInfo); ok {
		toName = name
	}
	evName := ""
	if name, ok := namedTypeName(callExpr.Args[2], pkg.TypesInfo); ok {
		evName = name
	}
	return Flow{
		From:             ctx.stateMachine,
		To:               toName,
		EventType:        evName,
		HandlerType:      ctx.kind,
		HandlerEventType: ctx.handlerEvent,
		HandlerID:        ctx.handlerID,
	}, true
}

func uniqueFlows(flows []Flow) []Flow {
	seen := make(map[string]bool)
	var unique []Flow
	for _, f := range flows {
		key := fmt.Sprintf("%s->%s:%s:%s:%s", f.From, f.To, f.EventType, f.HandlerType, f.HandlerEventType)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, f)
		}
	}
	return unique
}

func findNextFlows(f *Flow, flows []Flow) []Flow {
	var next []Flow
	for _, candidate := range flows {
		if candidate.HandlerType == onEventHandler &&
			candidate.HandlerEventType == f.EventType &&
			candidate.From == f.To {
			next = append(next, candidate)
		}
	}
	return next
}

func findFlowPosition(target *Flow, flows []Flow) int {
	for i, f := range flows {
		if f.HandlerID == target.HandlerID &&
			f.From == target.From &&
			f.To == target.To &&
			f.EventType == target.EventType {
			return i
		}
	}
	return len(flows)
}

func collectChain(f *Flow, flows []Flow, processed map[string]bool) []Flow {
	var chain []Flow
	for _, next := range findNextFlows(f, flows) {
		if processed[next.HandlerID] {
			continue
		}
		processed[next.HandlerID] = true
		chain = append(chain, next)
		chain = append(chain, collectChain(&next, flows, processed)...)
	}
	return chain
}

type elementBuilder struct {
	flows         []Flow
	processed     map[string]bool
	handlerGroups map[string][]Flow
	elements      []Element
}

func (b *elementBuilder) process(f *Flow) {
	if b.processed[f.HandlerID] {
		return
	}
	handlerFlows := b.handlerGroups[f.HandlerID]
	if len(handlerFlows) > 1 {
		var trigger Flow
		triggerFound := false
		for _, cand := range b.flows {
			if cand.EventType == f.HandlerEventType && cand.To == f.From {
				trigger = cand
				triggerFound = true
				break
			}
		}
		if triggerFound && !b.processed[trigger.HandlerID] {
			// Add the triggering flow as a normal element before the branches
			b.elements = append(b.elements, Element{Flows: []Flow{trigger}})
			b.processed[trigger.HandlerID] = true
		}
		sorted := append([]Flow(nil), handlerFlows...)
		sort.Slice(sorted, func(i, j int) bool {
			fi, fj := sorted[i], sorted[j]
			ci := len(findNextFlows(&fi, b.flows))
			cj := len(findNextFlows(&fj, b.flows))
			if ci != cj {
				return ci < cj
			}
			return findFlowPosition(&fi, b.flows) < findFlowPosition(&fj, b.flows)
		})
		var branchPaths [][]Flow
		for _, h := range sorted {
			path := append([]Flow{h}, collectChain(&h, b.flows, b.processed)...)
			branchPaths = append(branchPaths, path)
		}
		b.elements = append(b.elements, Element{Branches: branchPaths})
		b.processed[f.HandlerID] = true
		return
	}
	h := handlerFlows[0]
	b.elements = append(b.elements, Element{Flows: []Flow{h}})
	b.processed[h.HandlerID] = true
	for _, next := range findNextFlows(&h, b.flows) {
		nextCopy := next
		b.process(&nextCopy)
	}
}

func buildElements(flows []Flow) []Element {
	b := &elementBuilder{
		flows:         flows,
		processed:     make(map[string]bool),
		handlerGroups: make(map[string][]Flow),
	}
	for i := range flows {
		f := flows[i]
		b.handlerGroups[f.HandlerID] = append(b.handlerGroups[f.HandlerID], f)
	}
	for i := range flows {
		f := &flows[i]
		if f.HandlerType == onEntryHandler && !b.processed[f.HandlerID] {
			b.process(f)
		}
	}
	for i := range flows {
		f := &flows[i]
		if !b.processed[f.HandlerID] {
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
