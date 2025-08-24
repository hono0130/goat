package goat

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

// CommunicationFlow represents a single communication between state machines
type CommunicationFlow struct {
	From             string // source state machine (e.g., "Client")
	To               string // target state machine (e.g., "Server")
	EventType        string // event type being sent (e.g., "eCheckMenuExistenceRequest")
	HandlerType      string // "OnEntry", "OnEvent", "OnExit"
	HandlerEventType string // for OnEvent, the event type being handled
}

// HandlerInfo stores information about a goat handler registration
type HandlerInfo struct {
	StateMachineType string       // state machine type (e.g., "Client", "Server")
	Function         *ast.FuncLit // handler function AST node
	HandlerType      string       // "OnEntry", "OnEvent", "OnExit"
	EventType        string       // for OnEvent handlers, the event type being handled
}

// SendToInfo stores information about a goat.SendTo call
type SendToInfo struct {
	Target ast.Expr // target expression AST node
	Event  ast.Expr // event expression AST node
}

// AnalyzePackage analyzes a Go package and generates a Mermaid sequence diagram
func AnalyzePackage(packagePath, outputPath string) error {
	// 1. Load package with type information
	pkg, err := loadPackageWithTypes(packagePath)
	if err != nil {
		return fmt.Errorf("failed to load package with types: %w", err)
	}

	// 2. Extract state machine definition order
	stateMachineOrder := extractStateMachineOrder(pkg)

	// 3. Extract communication flows
	flows := extractCommunicationFlows(pkg)

	// 4. Generate Mermaid
	mermaidContent := generateMermaidWithOrder(flows, stateMachineOrder)

	// 5. Write to file
	return writeFile(outputPath, mermaidContent)
}

// loadPackageWithTypes loads a package with type information using go/packages
func loadPackageWithTypes(packagePath string) (*packages.Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
		Dir:  packagePath,
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

			if isStateMachineStruct(typeSpec) {
				name := typeSpec.Name.Name
				if !seenStateMachines[name] {
					stateMachineOrder = append(stateMachineOrder, name)
					seenStateMachines[name] = true
				}
			}
			return true
		})
	}
	
	return stateMachineOrder
}

// isStateMachineStruct checks if a type spec is a state machine struct
func isStateMachineStruct(typeSpec *ast.TypeSpec) bool {
	structType, ok := typeSpec.Type.(*ast.StructType)
	if !ok {
		return false
	}

	for _, field := range structType.Fields.List {
		if field.Names != nil { // not an embedded field
			continue
		}

		selExpr, ok := field.Type.(*ast.SelectorExpr)
		if !ok {
			continue
		}

		ident, ok := selExpr.X.(*ast.Ident)
		if !ok {
			continue
		}

		if ident.Name == "goat" && selExpr.Sel.Name == "StateMachine" {
			return true
		}
	}

	return false
}

// extractCommunicationFlows extracts all communication flows from the package
func extractCommunicationFlows(pkg *packages.Package) []CommunicationFlow {
	var flows []CommunicationFlow

	for _, file := range pkg.Syntax {
		handlers := findHandlerRegistrations(file, pkg)
		flows = append(flows, extractFlowsFromHandlers(handlers, pkg)...)
	}

	return buildRequestResponsePairs(flows)
}

// extractFlowsFromHandlers converts handlers to communication flows
func extractFlowsFromHandlers(handlers []HandlerInfo, pkg *packages.Package) []CommunicationFlow {
	var flows []CommunicationFlow

	for _, handler := range handlers {
		sendTos := findSendToCalls(handler.Function)
		for _, sendTo := range sendTos {
			flow := CommunicationFlow{
				From:             handler.StateMachineType,
				To:               resolveTargetType(sendTo.Target, pkg),
				EventType:        extractEventTypeFromExpr(sendTo.Event, pkg),
				HandlerType:      handler.HandlerType,
				HandlerEventType: handler.EventType,
			}
			flows = append(flows, flow)
		}
	}

	return flows
}

// findHandlerRegistrations finds all goat handler registrations in a file
func findHandlerRegistrations(file *ast.File, pkg *packages.Package) []HandlerInfo {
	var handlers []HandlerInfo

	ast.Inspect(file, func(n ast.Node) bool {
		callExpr, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		handler := extractHandlerInfo(callExpr, pkg)
		if handler != nil {
			handlers = append(handlers, *handler)
		}

		return true
	})

	return handlers
}

// extractHandlerInfo extracts handler information from a call expression
func extractHandlerInfo(callExpr *ast.CallExpr, pkg *packages.Package) *HandlerInfo {
	handlerType := getHandlerType(callExpr)
	if handlerType == "" {
		return nil
	}

	stateMachineType := extractStateMachineTypeFromSpec(callExpr, pkg)
	if stateMachineType == "" {
		return nil
	}

	if len(callExpr.Args) < 3 {
		return nil
	}

	handlerFunc, ok := callExpr.Args[len(callExpr.Args)-1].(*ast.FuncLit)
	if !ok {
		return nil
	}

	var eventType string
	if handlerType == "OnEvent" && len(callExpr.Args) >= 4 {
		eventType = extractEventTypeFromExpr(callExpr.Args[2], pkg)
	}

	return &HandlerInfo{
		StateMachineType: stateMachineType,
		Function:         handlerFunc,
		HandlerType:      handlerType,
		EventType:        eventType,
	}
}

// getHandlerType checks if a call expression is a goat handler registration
func getHandlerType(callExpr *ast.CallExpr) string {
	selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}

	if !isGoatSelector(selExpr) {
		return ""
	}

	switch selExpr.Sel.Name {
	case "OnEntry", "OnEvent", "OnExit":
		return selExpr.Sel.Name
	default:
		return ""
	}
}

// isGoatSelector checks if a selector expression is from the goat package
func isGoatSelector(selExpr *ast.SelectorExpr) bool {
	ident, ok := selExpr.X.(*ast.Ident)
	return ok && ident.Name == "goat"
}

// extractStateMachineTypeFromSpec extracts the state machine type from spec parameter using type information
func extractStateMachineTypeFromSpec(callExpr *ast.CallExpr, pkg *packages.Package) string {
	if len(callExpr.Args) == 0 {
		return ""
	}

	specType := getTypeOfExpression(callExpr.Args[0], pkg)
	if specType == nil {
		return ""
	}

	return extractTypeFromGeneric(specType)
}

// getTypeOfExpression gets the type of an AST expression
func getTypeOfExpression(expr ast.Expr, pkg *packages.Package) types.Type {
	tv, ok := pkg.TypesInfo.Types[expr]
	if !ok {
		return nil
	}
	return tv.Type
}

// extractTypeFromGeneric extracts the state machine type from a generic spec type
func extractTypeFromGeneric(typ types.Type) string {
	// Handle pointer to named type: *StateMachineSpec[*Type]
	if ptr, ok := typ.(*types.Pointer); ok {
		typ = ptr.Elem()
	}

	// Look for goat.StateMachineSpec[*StateMachineType] pattern
	named, ok := typ.(*types.Named)
	if !ok {
		return ""
	}

	typeArgs := named.TypeArgs()
	if typeArgs == nil || typeArgs.Len() == 0 {
		return ""
	}

	// Extract the first type argument: *StateMachineType
	arg := typeArgs.At(0)
	ptr, ok := arg.(*types.Pointer)
	if !ok {
		return ""
	}

	elem, ok := ptr.Elem().(*types.Named)
	if !ok {
		return ""
	}

	return elem.Obj().Name()
}

// findSendToCalls finds all goat.SendTo calls in a handler function
func findSendToCalls(handlerFunc *ast.FuncLit) []SendToInfo {
	var sendTos []SendToInfo

	ast.Inspect(handlerFunc.Body, func(n ast.Node) bool {
		callExpr, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		if isSendToCall(callExpr) && len(callExpr.Args) >= 3 {
			sendTos = append(sendTos, SendToInfo{
				Target: callExpr.Args[1],
				Event:  callExpr.Args[2],
			})
		}

		return true
	})

	return sendTos
}

// isSendToCall checks if a call expression is a goat.SendTo call
func isSendToCall(callExpr *ast.CallExpr) bool {
	selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	return isGoatSelector(selExpr) && selExpr.Sel.Name == "SendTo"
}

// resolveTargetType resolves target expressions to state machine names using type information
func resolveTargetType(targetExpr ast.Expr, pkg *packages.Package) string {
	typeName := extractTypeNameFromExpr(targetExpr, pkg)
	if typeName != "" {
		return typeName
	}
	return extractExpressionString(targetExpr)
}

// extractEventTypeFromExpr extracts event type name from event expression using type information
func extractEventTypeFromExpr(eventExpr ast.Expr, pkg *packages.Package) string {
	typeName := extractTypeNameFromExpr(eventExpr, pkg)
	if typeName != "" {
		return typeName
	}
	return extractEventTypeFromAST(eventExpr)
}

// extractTypeNameFromExpr extracts the type name from an expression
func extractTypeNameFromExpr(expr ast.Expr, pkg *packages.Package) string {
	typ := getTypeOfExpression(expr, pkg)
	if typ == nil {
		return ""
	}

	if ptr, ok := typ.(*types.Pointer); ok {
		typ = ptr.Elem()
	}

	if named, ok := typ.(*types.Named); ok {
		return named.Obj().Name()
	}

	return ""
}

// extractEventTypeFromAST extracts event type from AST expression (fallback)
func extractEventTypeFromAST(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.UnaryExpr:
		if e.Op == token.AND {
			if comp, ok := e.X.(*ast.CompositeLit); ok {
				if ident, ok := comp.Type.(*ast.Ident); ok {
					return ident.Name
				}
			}
		}
	case *ast.CompositeLit:
		if ident, ok := e.Type.(*ast.Ident); ok {
			return ident.Name
		}
	}
	return ""
}

// extractExpressionString converts an AST expression to a string representation (fallback)
func extractExpressionString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		base := extractExpressionString(e.X)
		if base != "" {
			return e.Sel.Name // Return just the field name for now
		}
		return e.Sel.Name
	}
	return "Unknown"
}

// buildRequestResponsePairs organizes flows into request-response pairs
func buildRequestResponsePairs(flows []CommunicationFlow) []CommunicationFlow {
	var orderedFlows []CommunicationFlow

	// 1. Find OnEntry flows (initiators)
	for _, flow := range flows {
		if flow.HandlerType == "OnEntry" {
			orderedFlows = append(orderedFlows, flow)

			// 2. Find corresponding OnEvent response
			response := findResponseFlow(flows, flow.EventType, flow.To)
			if response != nil {
				orderedFlows = append(orderedFlows, *response)
			}
		}
	}

	// 3. Add remaining flows that weren't paired
	for _, flow := range flows {
		if flow.HandlerType != "OnEntry" && !isFlowInList(flow, orderedFlows) {
			orderedFlows = append(orderedFlows, flow)
		}
	}

	return orderedFlows
}

// findResponseFlow finds a response flow for a given request
func findResponseFlow(flows []CommunicationFlow, requestEvent, responder string) *CommunicationFlow {
	for _, flow := range flows {
		if flow.HandlerType == "OnEvent" &&
			flow.HandlerEventType == requestEvent &&
			flow.From == responder {
			return &flow
		}
	}
	return nil
}

// isFlowInList checks if a flow is already in the list
func isFlowInList(flow CommunicationFlow, flows []CommunicationFlow) bool {
	for _, f := range flows {
		if f.From == flow.From && f.To == flow.To && f.EventType == flow.EventType &&
			f.HandlerType == flow.HandlerType && f.HandlerEventType == flow.HandlerEventType {
			return true
		}
	}
	return false
}

// generateMermaidWithOrder generates Mermaid sequence diagram syntax with ordered participants
func generateMermaidWithOrder(flows []CommunicationFlow, stateMachineOrder []string) string {
	participantsInFlows := collectParticipants(flows)
	orderedParticipants := filterOrderedParticipants(stateMachineOrder, participantsInFlows)
	return formatMermaidDiagram(orderedParticipants, flows)
}

// collectParticipants collects all participants from flows
func collectParticipants(flows []CommunicationFlow) map[string]bool {
	participants := make(map[string]bool)
	for _, flow := range flows {
		if flow.From != "" {
			participants[flow.From] = true
		}
		if flow.To != "" {
			participants[flow.To] = true
		}
	}
	return participants
}

// filterOrderedParticipants filters state machine order by actual participants
func filterOrderedParticipants(order []string, participants map[string]bool) []string {
	var filtered []string
	for _, sm := range order {
		if participants[sm] {
			filtered = append(filtered, sm)
		}
	}
	return filtered
}

// generateMermaid generates Mermaid sequence diagram syntax (kept for compatibility)
func generateMermaid(flows []CommunicationFlow) string {
	participantOrder := extractParticipantsInOrder(flows)
	return formatMermaidDiagram(participantOrder, flows)
}

// extractParticipantsInOrder extracts unique participants in order of appearance
func extractParticipantsInOrder(flows []CommunicationFlow) []string {
	var participantOrder []string
	seen := make(map[string]bool)
	
	for _, flow := range flows {
		if flow.From != "" && !seen[flow.From] {
			participantOrder = append(participantOrder, flow.From)
			seen[flow.From] = true
		}
		if flow.To != "" && !seen[flow.To] {
			participantOrder = append(participantOrder, flow.To)
			seen[flow.To] = true
		}
	}
	
	return participantOrder
}

// formatMermaidDiagram formats flows into Mermaid syntax
func formatMermaidDiagram(participants []string, flows []CommunicationFlow) string {
	var sb strings.Builder
	sb.WriteString("sequenceDiagram\n")

	for _, participant := range participants {
		sb.WriteString(fmt.Sprintf("    participant %s\n", participant))
	}

	for _, flow := range flows {
		if flow.From != "" && flow.To != "" && flow.EventType != "" {
			sb.WriteString(fmt.Sprintf("    %s->>%s: %s\n", flow.From, flow.To, flow.EventType))
		}
	}

	return sb.String()
}

// writeFile writes content to a file
func writeFile(outputPath, content string) error {
	// Create output directory if needed
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write content to file
	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}