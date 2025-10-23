package cmd

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/goatx/goat-cli/internal/test"
	"github.com/google/go-cmp/cmp"
)

func TestSequenceRenderCommand(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "sequence_diagram.mmd")
	args := []string{"render", "sequence", "--output", outputPath, test.FixtureDir(t)}

	origOut := rootCmd.OutOrStdout()
	origErr := rootCmd.ErrOrStderr()

	rootCmd.SetOut(io.Discard)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs(args)

	t.Cleanup(func() {
		rootCmd.SetArgs(nil)
		rootCmd.SetOut(origOut)
		rootCmd.SetErr(origErr)
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	got, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}

	want := test.ReadGolden(t, "sequence_diagram.golden")
	if diff := cmp.Diff(want, string(got)); diff != "" {
		t.Fatalf("mismatch (-want +got):\n%s", diff)
	}
}
