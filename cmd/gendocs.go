package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// This is a simplified version of your root command for documentation generation
// You may need to adjust this based on your actual command structure
func createRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "abaper",
		Short: "ABAP Development Tool - CLI and REST Services (No AI)",
		Long: `ABAP Development Tool - CLI and REST Services (No AI)

A comprehensive CLI tool for interacting with SAP ABAP systems via ADT.
Supports retrieving source code, searching objects, and testing connections.`,
		Version: "v0.0.1",
	}

	// Add subcommands
	serverCmd := &cobra.Command{
		Use:   "server",
		Short: "Run as REST API server",
		Long:  "Start the ABAPER REST API server for HTTP-based ABAP operations.",
	}

	getCmd := &cobra.Command{
		Use:   "get TYPE NAME [ARGS...]",
		Short: "Retrieve ABAP object source code",
		Long: `Retrieve ABAP object source code from SAP system.

TYPES:
  program     ABAP program/report
  class       ABAP class
  function    ABAP function module (requires function group)
  include     ABAP include
  interface   ABAP interface
  structure   ABAP structure
  table       ABAP table
  package     ABAP package contents

EXAMPLES:
  abaper get program ZTEST
  abaper get class ZCL_TEST
  abaper get function ZTEST_FUNC ZTEST_GROUP
  abaper get package $TMP`,
		Args: cobra.MinimumNArgs(2),
	}

	searchCmd := &cobra.Command{
		Use:   "search objects PATTERN [TYPES...]",
		Short: "Search for ABAP objects",
		Long: `Search for ABAP objects by pattern.

EXAMPLES:
  abaper search objects "Z*"
  abaper search objects "CL_*" class
  abaper search objects "*TEST*" program class`,
		Args: cobra.MinimumNArgs(2),
	}

	listCmd := &cobra.Command{
		Use:   "list TYPE [PATTERN]",
		Short: "List objects of specified type",
		Long: `List objects of specified type.

TYPES:
  packages    List packages

EXAMPLES:
  abaper list packages
  abaper list packages "Z*"`,
		Args: cobra.MinimumNArgs(1),
	}

	connectCmd := &cobra.Command{
		Use:   "connect",
		Short: "Test ADT connection",
		Long: `Test ADT connection to SAP system.

This command verifies:
- Basic connectivity to SAP system
- ADT service availability
- Authentication credentials
- User permissions`,
	}

	// Add flags to match your actual implementation
	rootCmd.PersistentFlags().BoolP("quiet", "q", true, "Quiet mode (DEFAULT - minimal CLI output)")
	rootCmd.PersistentFlags().Bool("normal", false, "Normal mode (show standard output)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Verbose mode (detailed output + debug info)")
	rootCmd.PersistentFlags().String("log-file", "", "Log to specified file (auto-creates directory)")
	rootCmd.PersistentFlags().String("adt-host", "", "SAP system host (or set SAP_HOST)")
	rootCmd.PersistentFlags().String("adt-client", "", "SAP client (or set SAP_CLIENT)")
	rootCmd.PersistentFlags().String("adt-username", "", "SAP username (or set SAP_USERNAME)")
	rootCmd.PersistentFlags().String("adt-password", "", "SAP password (or set SAP_PASSWORD)")
	rootCmd.PersistentFlags().String("config", "", "Configuration file path")

	serverCmd.Flags().StringP("port", "p", "8080", "Port for server mode")

	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(connectCmd)

	return rootCmd
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: gendocs <man|markdown|yaml|rest>")
		os.Exit(1)
	}

	docType := os.Args[1]
	rootCmd := createRootCmd()

	// Create output directories
	docsDir := "docs"
	manDir := filepath.Join(docsDir, "man")
	mdDir := filepath.Join(docsDir, "markdown")
	yamlDir := filepath.Join(docsDir, "yaml")
	restDir := filepath.Join(docsDir, "rest")

	// Ensure directories exist
	for _, dir := range []string{docsDir, manDir, mdDir, yamlDir, restDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	switch docType {
	case "man":
		fmt.Println("Generating man pages...")
		if err := doc.GenManTree(rootCmd, &doc.GenManHeader{
			Title:   "ABAPER",
			Section: "1",
			Manual:  "ABAPER Manual",
			Source:  "ABAPER v0.0.1",
		}, manDir); err != nil {
			log.Fatalf("Failed to generate man pages: %v", err)
		}
		fmt.Printf("Man pages generated in %s/\n", manDir)

	case "markdown":
		fmt.Println("Generating markdown documentation...")
		if err := doc.GenMarkdownTree(rootCmd, mdDir); err != nil {
			log.Fatalf("Failed to generate markdown docs: %v", err)
		}
		fmt.Printf("Markdown documentation generated in %s/\n", mdDir)

	case "yaml":
		fmt.Println("Generating YAML documentation...")
		if err := doc.GenYamlTree(rootCmd, yamlDir); err != nil {
			log.Fatalf("Failed to generate YAML docs: %v", err)
		}
		fmt.Printf("YAML documentation generated in %s/\n", yamlDir)

	case "rest":
		fmt.Println("Generating reStructuredText documentation...")
		if err := doc.GenReSTTree(rootCmd, restDir); err != nil {
			log.Fatalf("Failed to generate ReST docs: %v", err)
		}
		fmt.Printf("reStructuredText documentation generated in %s/\n", restDir)

	case "all":
		fmt.Println("Generating all documentation formats...")
		
		// Generate man pages
		if err := doc.GenManTree(rootCmd, &doc.GenManHeader{
			Title:   "ABAPER",
			Section: "1",
			Manual:  "ABAPER Manual",
			Source:  "ABAPER v0.0.1",
		}, manDir); err != nil {
			log.Fatalf("Failed to generate man pages: %v", err)
		}
		
		// Generate markdown
		if err := doc.GenMarkdownTree(rootCmd, mdDir); err != nil {
			log.Fatalf("Failed to generate markdown docs: %v", err)
		}
		
		// Generate YAML
		if err := doc.GenYamlTree(rootCmd, yamlDir); err != nil {
			log.Fatalf("Failed to generate YAML docs: %v", err)
		}
		
		// Generate ReST
		if err := doc.GenReSTTree(rootCmd, restDir); err != nil {
			log.Fatalf("Failed to generate ReST docs: %v", err)
		}
		
		fmt.Printf("All documentation formats generated in %s/\n", docsDir)

	default:
		fmt.Printf("Unknown documentation type: %s\n", docType)
		fmt.Println("Available types: man, markdown, yaml, rest, all")
		os.Exit(1)
	}
}
