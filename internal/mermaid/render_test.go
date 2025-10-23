package mermaid

import (
	"bytes"
	"testing"

	"github.com/goatx/goat-cli/internal/test"
	"github.com/google/go-cmp/cmp"
)

func TestRenderSequenceDiagram(t *testing.T) {
	t.Parallel()
	pkg := loadSpecPackage(t)
	var buf bytes.Buffer
	if err := RenderSequenceDiagram(pkg, &buf); err != nil {
		t.Fatalf("RenderSequenceDiagram returned error: %v", err)
	}

	got := buf.String()
	want := test.ReadGolden(t, "sequence_diagram.golden")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("rendered diagram mismatch (-want +got):\n%s", diff)
	}
}
