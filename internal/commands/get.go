package commands

import (
	"fmt"

	"github.com/bluefunda/abaper/internal/client"
	"github.com/bluefunda/abaper/pkg/output"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Fetch an ABAP object's source from the target system",
	RunE:  runGet,
}

func init() {
	getCmd.Flags().String("name", "", "Object name (required)")
	getCmd.Flags().String("type", "program", "Object type: program, class, interface")
	_ = getCmd.MarkFlagRequired("name")
}

func runGet(cmd *cobra.Command, args []string) error {
	objectType, _ := cmd.Flags().GetString("type")
	objectName, _ := cmd.Flags().GetString("name")

	c, err := client.NewClient()
	if err != nil {
		return err
	}

	obj, err := c.GetObject(objectType, objectName)
	if err != nil {
		return fmt.Errorf("get object: %w", err)
	}

	outputFmt, _ := cmd.Flags().GetString("output")
	if outputFmt == "json" {
		output.PrintJSON(*obj)
		return nil
	}

	source, _ := (*obj)["source"].(string)
	if source == "" {
		return fmt.Errorf("no source found for %s %s", objectType, objectName)
	}
	fmt.Println(source)
	return nil
}
