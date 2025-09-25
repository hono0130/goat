package mermaid

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// Generate analyzes a Go package and writes a Mermaid sequence diagram.
// It inspects state machines and their communication flows to produce a
// Mermaid `sequenceDiagram` definition.
//
// Parameters:
//   - packagePath: File system path to the target Go package directory
//   - writer: Destination io.Writer that receives the generated Mermaid output
//
// Example:
//
//	var buf bytes.Buffer
//	if err := mermaid.Generate(".", &buf); err != nil {
//		log.Fatal(err)
//	}
//	fmt.Print(buf.String())
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

func render(elements []element, order []string) string {
	participants := orderedParticipants(elements, order)
	var sb strings.Builder
	sb.WriteString("sequenceDiagram\n")
	for _, p := range participants {
		sb.WriteString(fmt.Sprintf("    participant %s\n", p))
	}
	sb.WriteString("\n")
	for _, e := range elements {
		if len(e.branches) > 0 {
			for i, br := range e.branches {
				if i == 0 {
					sb.WriteString("    alt\n")
				} else {
					sb.WriteString("    else\n")
				}
				for _, f := range br {
					sb.WriteString(fmt.Sprintf("        %s->>%s: %s\n", f.from, f.to, f.eventType))
				}
			}
			sb.WriteString("    end\n")
		} else {
			for _, f := range e.flows {
				sb.WriteString(fmt.Sprintf("    %s->>%s: %s\n", f.from, f.to, f.eventType))
			}
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
		if len(e.branches) > 0 {
			for _, br := range e.branches {
				for _, f := range br {
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
		}
	}
	sort.Strings(extras)
	participants = append(participants, extras...)
	return participants
}
