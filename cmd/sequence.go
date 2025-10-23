/*
Copyright © 2025 Honoka Toda, Shinya Ishitobi

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"os"

	"github.com/goatx/goat-cli/internal/load"
	"github.com/goatx/goat-cli/internal/mermaid"
	"github.com/spf13/cobra"
)

// sequenceCmd represents the sequence command
var sequenceCmd = &cobra.Command{
	Use:   "sequence",
	Short: "Generate a Mermaid sequence diagram",
	Long: `Analyze a Go package that includes goat state machines and emit a Mermaid sequenceDiagram definition.
Provide the target package path as the argument and write the result to stdout or to a file via -o/--output.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		outputPath, err := cmd.Flags().GetString("output")
		if err != nil {
			return err
		}

		pkg, err := load.Load(args[0])
		if err != nil {
			return err
		}

		writer := cmd.OutOrStdout()

		if outputPath != "" {
			file, err := os.Create(outputPath)
			if err != nil {
				return err
			}
			defer file.Close()
			writer = file
		}

		err = mermaid.RenderSequenceDiagram(pkg, writer)
		if err != nil {
			return err
		}
		return nil
	},
}

func init() {
	renderCmd.AddCommand(sequenceCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// sequenceCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// sequenceCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	sequenceCmd.Flags().StringP("output", "o", "", "write the generated diagram to a file")
}
