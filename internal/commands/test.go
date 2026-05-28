package commands

import (
	"fmt"

	"github.com/bluefunda/abaper/internal/client"
	"github.com/bluefunda/abaper/pkg/output"
	"github.com/spf13/cobra"
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Run ABAP unit tests for an object",
	RunE:  runTest,
}

func init() {
	testCmd.Flags().String("type", "class", "Object type: class, program")
	testCmd.Flags().String("name", "", "Object name (required)")
	_ = testCmd.MarkFlagRequired("name")
}

func runTest(cmd *cobra.Command, args []string) error {
	objectType, _ := cmd.Flags().GetString("type")
	objectName, _ := cmd.Flags().GetString("name")

	c, err := client.NewClient()
	if err != nil {
		return err
	}

	fmt.Printf("Running unit tests for %s %s...\n", objectType, objectName)
	result, err := c.RunUnitTests(objectName, objectType)
	if err != nil {
		return fmt.Errorf("unit tests failed: %w", err)
	}

	outputFmt, _ := cmd.Flags().GetString("output")
	if outputFmt == "json" {
		output.PrintJSON(result)
	} else {
		output.PrintJSON(result)
	}

	return nil
}
