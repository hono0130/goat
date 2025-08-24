package main

import (
	"fmt"
	"os"

	goat "github.com/goatx/goat"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <package-path> <output-file>\n", os.Args[0])
		os.Exit(1)
	}

	packagePath := os.Args[1]
	outputPath := os.Args[2]

	if err := goat.AnalyzePackage(packagePath, outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Mermaid sequence diagram generated: %s\n", outputPath)
}