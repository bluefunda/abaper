package commands

import (
	"fmt"
	"strings"

	"github.com/bluefunda/abaper/internal/client"
	"github.com/bluefunda/abaper/pkg/output"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List ABAP objects or package contents",
}

var listObjectsCmd = &cobra.Command{
	Use:   "objects",
	Short: "List ABAP objects, optionally filtered by package or type",
	RunE:  runListObjects,
}

var listPackagesCmd = &cobra.Command{
	Use:   "packages",
	Short: "List contents of an ABAP package",
	RunE:  runListPackages,
}

func init() {
	listObjectsCmd.Flags().String("package", "", "Filter by package name")
	listObjectsCmd.Flags().String("type", "", "Filter by object type")

	listPackagesCmd.Flags().String("name", "", "Package name (required)")
	_ = listPackagesCmd.MarkFlagRequired("name")

	listCmd.AddCommand(listObjectsCmd)
	listCmd.AddCommand(listPackagesCmd)
}

func runListObjects(cmd *cobra.Command, args []string) error {
	packageName, _ := cmd.Flags().GetString("package")
	objectType, _ := cmd.Flags().GetString("type")

	c, err := client.NewClient()
	if err != nil {
		return err
	}

	objects, err := c.ListObjects(packageName, objectType)
	if err != nil {
		return fmt.Errorf("list objects failed: %w", err)
	}

	outputFmt, _ := cmd.Flags().GetString("output")
	if outputFmt == "json" {
		output.PrintJSON(objects)
	} else {
		if len(objects) == 0 {
			fmt.Println("No objects found.")
			return nil
		}
		for _, obj := range objects {
			parts := []string{}
			if t, ok := obj["type"]; ok {
				parts = append(parts, fmt.Sprintf("%v", t))
			}
			if n, ok := obj["name"]; ok {
				parts = append(parts, fmt.Sprintf("%v", n))
			}
			fmt.Println(strings.Join(parts, "\t"))
		}
	}

	return nil
}

func runListPackages(cmd *cobra.Command, args []string) error {
	packageName, _ := cmd.Flags().GetString("name")

	c, err := client.NewClient()
	if err != nil {
		return err
	}

	objects, err := c.PackageContents(packageName)
	if err != nil {
		return fmt.Errorf("package contents failed: %w", err)
	}

	outputFmt, _ := cmd.Flags().GetString("output")
	if outputFmt == "json" {
		output.PrintJSON(objects)
	} else {
		if len(objects) == 0 {
			fmt.Println("No objects found in package.")
			return nil
		}
		for _, obj := range objects {
			parts := []string{}
			if t, ok := obj["type"]; ok {
				parts = append(parts, fmt.Sprintf("%v", t))
			}
			if n, ok := obj["name"]; ok {
				parts = append(parts, fmt.Sprintf("%v", n))
			}
			fmt.Println(strings.Join(parts, "\t"))
		}
	}

	return nil
}
