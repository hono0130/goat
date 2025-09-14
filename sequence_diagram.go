package goat

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
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

// CommunicationFlow represents a single communication between state machines
type CommunicationFlow struct {
	From             string // source state machine
	To               string // target state machine
	EventType        string // event type being sent
	HandlerType      string // "OnEntry", "OnEvent", "OnExit"
	HandlerEventType string // for OnEvent, the event type being handled
	HandlerID        string // identifies the handler for grouping flows
}

// HandlerInfo stores information about a goat handler registration
type HandlerInfo struct {
	StateMachineType string       // state machine type
	Function         *ast.FuncLit // handler function AST node
	HandlerType      string       // "OnEntry", "OnEvent", "OnExit"
	EventType        string       // for OnEvent handlers, the event type being handled
}

// SendToInfo stores information about a goat.SendTo call
type SendToInfo struct {
	Target ast.Expr // target expression AST node
	Event  ast.Expr // event expression AST node
}

// SequenceDiagramElement represents an element in the sequence diagram
type SequenceDiagramElement struct {
	Flows      []CommunicationFlow
	IsOptional bool // whether to display as opt block
}

// AnalyzePackage analyzes a Go package and generates a Mermaid sequence diagram
func AnalyzePackage(packagePath string, writer io.Writer) error {
	// Load package with type information
	pkg, err := loadPackageWithTypes(packagePath)
	if err != nil {
		return fmt.Errorf("failed to load package with types: %w", err)
	}

	// Extract state machine definition order
	stateMachineOrder := extractStateMachineOrder(pkg)

	// Extract communication flows
	flows := extractCommunicationFlows(pkg)

	// Build sequence diagram elements with opt blocks
	elements := buildSequenceDiagramElements(flows)

	// Generate Mermaid with groups
	mermaidContent := generateMermaidWithGroups(elements, stateMachineOrder)

	// Write to writer
	_, err = writer.Write([]byte(mermaidContent))
	if err != nil {
		return fmt.Errorf("failed to write content: %w", err)
	}

	return nil
}

// loadPackageWithTypes loads a package with type information using go/packages
func loadPackageWithTypes(packagePath string) (*packages.Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesSizes |
			packages.NeedTypesInfo |
			packages.NeedImports |
			packages.NeedDeps |
			packages.NeedModule,
		Dir: packagePath,
	}
	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		return nil, fmt.Errorf("failed to load package: %w", err)
	}

	if len(pkgs) != 1 {
		return nil, fmt.Errorf("expected 1 package, got %d", len(pkgs))
	}

	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		return nil, fmt.Errorf("package has errors: %v", pkg.Errors)
	}

	return pkg, nil
}

// extractStateMachineOrder extracts state machine types in definition order
func extractStateMachineOrder(pkg *packages.Package) []string {
	var stateMachineOrder []string
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
						stateMachineOrder = append(stateMachineOrder, name)
						seenStateMachines[name] = true
					}
					break
				}
			}
			return true
		})
	}

	return stateMachineOrder
}

// extractCommunicationFlows extracts all communication flows from the package
func extractCommunicationFlows(pkg *packages.Package) []CommunicationFlow {
	var flows []CommunicationFlow

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

				// Extract flow information
				flow := CommunicationFlow{
					From:             stateMachineType,
					To:               resolveTargetType(sendToCall.Args[1], pkg),
					EventType:        getEventType(sendToCall.Args[2], pkg),
					HandlerType:      handlerType,
					HandlerEventType: eventType,
					HandlerID:        handlerID,
				}
				flows = append(flows, flow)

				return true
			})

			return true
		})
	}

	// Remove duplicates
	seen := make(map[string]bool)
	var uniqueFlows []CommunicationFlow

	for _, flow := range flows {
		// Create a key for deduplication that includes handler information to preserve conditional branches
		key := fmt.Sprintf("%s->%s:%s:%s:%s", flow.From, flow.To, flow.EventType, flow.HandlerType, flow.HandlerEventType)

		if !seen[key] {
			seen[key] = true
			uniqueFlows = append(uniqueFlows, flow)
		}
	}

	return uniqueFlows
}

// isFromGoat checks if a selector expression is from the goat package using type information
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

// getTypeName extracts type name from expression with type info priority and AST fallback
func getTypeName(expr ast.Expr, pkg *packages.Package, isEvent bool) string {
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

// resolveTargetType resolves target expressions to state machine names using type information
func resolveTargetType(targetExpr ast.Expr, pkg *packages.Package) string {
	return getTypeName(targetExpr, pkg, false)
}

// getEventType extracts event type name from event expression using type information
func getEventType(eventExpr ast.Expr, pkg *packages.Package) string {
	return getTypeName(eventExpr, pkg, true)
}

// buildSequenceDiagramElements organizes flows into sequence diagram elements with opt blocks
func buildSequenceDiagramElements(flows []CommunicationFlow) []SequenceDiagramElement {
	var elements []SequenceDiagramElement
	processed := make(map[string]bool)
	handlerGroups := make(map[string][]CommunicationFlow)

	// Group flows by handler ID
	for _, flow := range flows {
		handlerGroups[flow.HandlerID] = append(handlerGroups[flow.HandlerID], flow)
	}

	// Helper functions (inlined for better readability)
	findTriggerFlow := func(handlerFlow CommunicationFlow) *CommunicationFlow {
		for _, flow := range flows {
			if flow.EventType == handlerFlow.HandlerEventType && flow.To == handlerFlow.From {
				return &flow
			}
		}
		return nil
	}

	findNextFlows := func(flow CommunicationFlow) []CommunicationFlow {
		var next []CommunicationFlow
		for _, candidate := range flows {
			if candidate.HandlerType == onEventHandler &&
				candidate.HandlerEventType == flow.EventType &&
				candidate.From == flow.To {
				next = append(next, candidate)
			}
		}
		return next
	}

	findFlowPosition := func(target CommunicationFlow) int {
		for i, flow := range flows {
			if flow.HandlerID == target.HandlerID &&
				flow.From == target.From &&
				flow.To == target.To &&
				flow.EventType == target.EventType {
				return i
			}
		}
		return len(flows)
	}

	var collectChain func(CommunicationFlow) []CommunicationFlow
	collectChain = func(flow CommunicationFlow) []CommunicationFlow {
		var chain []CommunicationFlow
		for _, next := range findNextFlows(flow) {
			if !processed[next.HandlerID] {
				chain = append(chain, next)
				processed[next.HandlerID] = true
				chain = append(chain, collectChain(next)...)
			}
		}
		return chain
	}

	var processFlow func(CommunicationFlow) []SequenceDiagramElement
	processFlow = func(flow CommunicationFlow) []SequenceDiagramElement {
		var elements []SequenceDiagramElement

		if processed[flow.HandlerID] {
			return elements
		}

		handlerFlows := handlerGroups[flow.HandlerID]

		if len(handlerFlows) > 1 {
			// Multiple flows from same handler = conditional branches
			// Put trigger flow in opt block first if needed
			if trigger := findTriggerFlow(flow); trigger != nil && !processed[trigger.HandlerID] {
				elements = append(elements, SequenceDiagramElement{
					Flows:      []CommunicationFlow{*trigger},
					IsOptional: true,
				})
				processed[trigger.HandlerID] = true
			}

			// Sort flows by original order to ensure consistent output
			sortedFlows := make([]CommunicationFlow, len(handlerFlows))
			copy(sortedFlows, handlerFlows)
			sort.Slice(sortedFlows, func(i, j int) bool {
				flow1, flow2 := sortedFlows[j], sortedFlows[i]
				pos1, pos2 := findFlowPosition(flow1), findFlowPosition(flow2)
				hasChain1 := len(findNextFlows(flow1)) > 0
				hasChain2 := len(findNextFlows(flow2)) > 0

				if hasChain1 && !hasChain2 {
					return true
				}
				if !hasChain1 && hasChain2 {
					return false
				}
				return pos1 > pos2
			})

			// Each conditional path becomes an opt block
			for _, hFlow := range sortedFlows {
				pathFlows := []CommunicationFlow{hFlow}
				pathFlows = append(pathFlows, collectChain(hFlow)...)

				elements = append(elements, SequenceDiagramElement{
					Flows:      pathFlows,
					IsOptional: true,
				})
			}
			processed[flow.HandlerID] = true
		} else {
			// Single flow
			hFlow := handlerFlows[0]
			elements = append(elements, SequenceDiagramElement{
				Flows:      []CommunicationFlow{hFlow},
				IsOptional: false,
			})
			processed[hFlow.HandlerID] = true

			// Process next flows
			for _, next := range findNextFlows(hFlow) {
				elements = append(elements, processFlow(next)...)
			}
		}

		return elements
	}

	// Start with OnEntry flows and follow the chain
	for _, flow := range flows {
		if flow.HandlerType == onEntryHandler && !processed[flow.HandlerID] {
			elements = append(elements, processFlow(flow)...)
		}
	}

	// Process remaining flows
	for _, flow := range flows {
		if !processed[flow.HandlerID] {
			elements = append(elements, processFlow(flow)...)
		}
	}

	return elements
}

// generateMermaidWithGroups generates Mermaid sequence diagram with opt block support
func generateMermaidWithGroups(elements []SequenceDiagramElement, stateMachineOrder []string) string {
	var sb strings.Builder
	sb.WriteString("sequenceDiagram\n")

	// Collect all participants from flows
	allParticipants := make(map[string]bool)
	for _, participant := range stateMachineOrder {
		allParticipants[participant] = true
	}

	// Add any participants that appear in flows but not in definition order
	var additionalParticipants []string
	for _, element := range elements {
		for _, flow := range element.Flows {
			if !allParticipants[flow.From] {
				additionalParticipants = append(additionalParticipants, flow.From)
				allParticipants[flow.From] = true
			}
			if !allParticipants[flow.To] {
				additionalParticipants = append(additionalParticipants, flow.To)
				allParticipants[flow.To] = true
			}
		}
	}

	// Sort additional participants for consistent output
	sort.Strings(additionalParticipants)

	// Write participants in definition order first
	for _, participant := range stateMachineOrder {
		sb.WriteString(fmt.Sprintf("    participant %s\n", participant))
	}

	// Write additional participants found in flows
	for _, participant := range additionalParticipants {
		sb.WriteString(fmt.Sprintf("    participant %s\n", participant))
	}

	// Add blank line after participants
	sb.WriteString("\n")

	// Write flows
	for _, element := range elements {
		if element.IsOptional {
			sb.WriteString("    opt\n")
			for _, f := range element.Flows {
				sb.WriteString(fmt.Sprintf("        %s->>%s: %s\n",
					f.From, f.To, f.EventType))
			}
			sb.WriteString("    end\n")
		} else {
			for _, f := range element.Flows {
				sb.WriteString(fmt.Sprintf("    %s->>%s: %s\n",
					f.From, f.To, f.EventType))
			}
		}
	}

	return sb.String()
}
