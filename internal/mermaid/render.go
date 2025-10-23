package mermaid

import (
	"fmt"
	"io"
	"strings"

	"github.com/goatx/goat-cli/internal/load"
)

// RenderSequenceDiagram analyzes a Go package and writes a Mermaid sequence diagram.
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
//	if err := mermaid.RenderSequenceDiagram(".", &buf); err != nil {
//		log.Fatal(err)
//	}
func RenderSequenceDiagram(pkg *load.PackageInfo, writer io.Writer) error {
	sequenceDiagram, err := analyze(pkg)
	if err != nil {
		return err
	}

	var sb strings.Builder
	sb.WriteString("sequenceDiagram\n")

	for _, p := range sequenceDiagram.participants {
		sb.WriteString(fmt.Sprintf("    participant %s\n", p))
	}
	sb.WriteString("\n")

	renderElements(&sb, sequenceDiagram.elements, 1)

	_, err = writer.Write([]byte(sb.String()))

	return err
}

func renderElements(sb *strings.Builder, elements []element, indent int) {
	for _, e := range elements {

		if len(e.branches) == 0 {
			writeFlow(sb, e.flow, indent)
			continue
		}

		writeIndent(sb, indent)
		sb.WriteString("alt\n")
		for i, br := range e.branches {
			if i > 0 {
				writeIndent(sb, indent)
				sb.WriteString("else\n")
			}
			writeFlow(sb, br.flow, indent+1)
			renderElements(sb, br.elements, indent+1)
		}
		writeIndent(sb, indent)
		sb.WriteString("end\n")
	}
}

func writeFlow(sb *strings.Builder, f flow, indent int) {
	writeIndent(sb, indent)
	sb.WriteString(fmt.Sprintf("%s->>%s: %s\n", f.from, f.to, f.eventType))
}

func writeIndent(sb *strings.Builder, indent int) {
	for i := 0; i < indent; i++ {
		sb.WriteString("    ")
	}
}
