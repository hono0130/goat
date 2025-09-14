package mermaid

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

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
