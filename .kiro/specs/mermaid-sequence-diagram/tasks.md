# Implementation Plan

- [ ] 1. Create sequence_diagram.go file structure

  - Create `sequence_diagram.go` in the goat root directory
  - Define data structures (`CommunicationFlow`, `HandlerInfo`, `SendToInfo`, `EventHandlerMapping`)
  - Create skeleton functions with empty implementations
  - _Requirements: 1.1, 1.3_

- [ ] 2. Implement AST parsing for Go files

  - Implement `parseGoFiles(packagePath string) ([]*ast.File, error)`
  - Parse all .go files in the package directory
  - Return AST nodes with proper error handling
  - _Requirements: 1.1_

- [ ] 3. Extract goat handler registrations from AST

  - Implement `collectInitialHandlers(files []*ast.File) []HandlerInfo`
  - Implement `collectAndIndexOnEventHandlers(files []*ast.File) map[string][]EventHandlerMapping`
  - Find `goat.OnEntry`, `goat.OnTransition`, `goat.OnExit`, `goat.OnHalt` calls for initial handlers
  - Find `goat.OnEvent` calls and index by event type and state machine
  - Extract state machine type from spec parameter's type (e.g., `*goat.Spec[*Client]`)
  - Store handler function AST nodes with their types
  - _Requirements: 1.1, 2.1_

- [ ] 4. Extract SendTo calls from handler functions

  - Implement `findSendToCalls(handlerFunc *ast.FuncLit) []SendToInfo`
  - Walk handler function AST to find `goat.SendTo` calls
  - Extract target and event expressions
  - _Requirements: 1.2, 2.1_

- [ ] 5. Implement name resolution functions

  - Implement `resolveTargetName(targetExpr string) string`
  - Implement `extractEventType(eventExpr string) string`
  - Handle patterns like `client.Server`, `event.From`, `&eCheckMenuExistenceRequest{}`
  - _Requirements: 2.2_

- [ ] 6. Build communication flows and trace event chains

  - Implement `extractCommunicationFlows(files []*ast.File) []CommunicationFlow`
  - Implement `traceEventChainFromSendTo(sendTo SendToInfo, fromMachine string, eventHandlers map[string][]EventHandlerMapping, visited map[string]bool) []CommunicationFlow`
  - Start from initial handlers (OnEntry/OnTransition/OnExit/OnHalt) with SendTo calls
  - For each SendTo, find matching OnEvent handlers by target machine and event type
  - Recursively trace SendTo calls within matched OnEvent handlers
  - Create CommunicationFlow structs with resolved names
  - Prevent infinite loops with visited tracking
  - _Requirements: 1.4, 2.2_

- [ ] 7. ~~Implement request-response pairing~~ (Not needed - event chain tracing handles ordering)

  - ~~Event chain tracing in step 6 already handles proper chronological ordering~~
  - ~~No separate pairing step needed as chains are traced naturally~~
  - _Requirements: 3.1, 3.2_

- [ ] 8. Generate Mermaid sequence diagram

  - Implement `generateMermaid(flows []CommunicationFlow) string`
  - Extract unique participants
  - Generate sequenceDiagram header and participant declarations
  - Generate message arrows with proper formatting
  - _Requirements: 1.3, 4.3_

- [ ] 9. Implement file output

  - Implement `writeFile(outputPath, content string) error`
  - Create output directory if needed
  - Write Mermaid content with proper error handling
  - _Requirements: 4.3_

- [ ] 10. Create public API function

  - Implement `AnalyzePackage(packagePath, outputPath string) error`
  - Integrate all components in the correct sequence
  - Add validation and comprehensive error reporting
  - _Requirements: 1.1, 1.2, 1.3, 1.4_

- [ ] 11. Write unit tests

  - Create `sequence_diagram_test.go`
  - Test AST parsing with sample Go code
  - Test communication flow extraction
  - Test Mermaid generation
  - _Requirements: All_

- [ ] 12. Test with client-server example

  - Run analysis on `example/client-server/main.go`
  - Verify generated diagram shows Client→Server event chains:
    - OnEntry: Client→Server: eCheckMenuExistenceRequest
    - OnEvent: Server→Client: eCheckMenuExistenceResponse (multiple variants)
  - Ensure event types are correctly extracted
  - Verify multiple OnEvent handlers for same event are all captured
  - _Requirements: 2.1, 2.2, 3.1, 3.2_

- [ ] 13. Handle edge cases and errors

  - Add graceful handling for unresolvable targets
  - Handle missing or malformed AST nodes
  - Provide informative error messages
  - _Requirements: 3.3, 4.1, 4.2_