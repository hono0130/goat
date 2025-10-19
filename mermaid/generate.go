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
	diagram, err := Analyze(packagePath)
	if err != nil {
		return err
	}
	_, err = writer.Write([]byte(render(diagram)))
	return err
}

func render(diagram *Diagram) string {
	var sb strings.Builder
	sb.WriteString("sequenceDiagram\n")
	for _, p := range diagram.Participants {
		sb.WriteString(fmt.Sprintf("    participant %s\n", p))
	}
	sb.WriteString("\n")
	for _, e := range diagram.Elements {
		if len(e.Branches) > 0 {
			for i, br := range e.Branches {
				if i == 0 {
					sb.WriteString("    alt\n")
				} else {
					sb.WriteString("    else\n")
				}
				for _, f := range br {
					sb.WriteString(fmt.Sprintf("        %s->>%s: %s\n", f.From, f.To, f.EventType))
				}
			}
			sb.WriteString("    end\n")
		} else {
			for _, f := range e.Flows {
				sb.WriteString(fmt.Sprintf("    %s->>%s: %s\n", f.From, f.To, f.EventType))
			}
		}
	}
	return sb.String()
}

func orderedParticipants(elements []Element, order []string) []string {
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
		for _, f := range e.Flows {
			if !seen[f.From] {
				extras = append(extras, f.From)
				seen[f.From] = true
			}
			if !seen[f.To] {
				extras = append(extras, f.To)
				seen[f.To] = true
			}
		}
		if len(e.Branches) > 0 {
			for _, br := range e.Branches {
				for _, f := range br {
					if !seen[f.From] {
						extras = append(extras, f.From)
						seen[f.From] = true
					}
					if !seen[f.To] {
						extras = append(extras, f.To)
						seen[f.To] = true
					}
				}
			}
		}
	}
	sort.Strings(extras)
	participants = append(participants, extras...)
	return participants
}
