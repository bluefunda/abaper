package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// Add this to your main.go file or create a separate docs command

// docsCmd represents the docs command
var docsCmd = &cobra.Command{
	Use:   "docs [FORMAT]",
	Short: "Generate documentation in various formats",
	Long: `Generate documentation for the abaper CLI tool in various formats.

Available formats:
  man        Generate man pages
  markdown   Generate markdown documentation
  yaml       Generate YAML documentation
  rest       Generate reStructuredText documentation
  all        Generate all formats (default)

Examples:
  abaper docs              # Generate all formats
  abaper docs markdown     # Generate only markdown
  abaper docs man          # Generate only man pages`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		format := "all"
		if len(args) > 0 {
			format = args[0]
		}

		outputDir := "docs"
		if dir, _ := cmd.Flags().GetString("output"); dir != "" {
			outputDir = dir
		}

		return generateDocumentation(format, outputDir)
	},
}

func generateDocumentation(format, outputDir string) error {
	// Create output directories
	manDir := filepath.Join(outputDir, "man")
	mdDir := filepath.Join(outputDir, "markdown")
	yamlDir := filepath.Join(outputDir, "yaml")
	restDir := filepath.Join(outputDir, "rest")

	// Ensure directories exist
	for _, dir := range []string{outputDir, manDir, mdDir, yamlDir, restDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}

	switch format {
	case "man":
		fmt.Println("Generating man pages...")
		if err := doc.GenManTree(rootCmd, &doc.GenManHeader{
			Title:   "ABAPER",
			Section: "1",
			Manual:  "ABAPER Manual",
			Source:  fmt.Sprintf("ABAPER %s", Version),
		}, manDir); err != nil {
			return fmt.Errorf("failed to generate man pages: %v", err)
		}
		fmt.Printf("Man pages generated in %s/\n", manDir)

	case "markdown":
		fmt.Println("Generating markdown documentation...")
		if err := doc.GenMarkdownTree(rootCmd, mdDir); err != nil {
			return fmt.Errorf("failed to generate markdown docs: %v", err)
		}
		fmt.Printf("Markdown documentation generated in %s/\n", mdDir)

	case "yaml":
		fmt.Println("Generating YAML documentation...")
		if err := doc.GenYamlTree(rootCmd, yamlDir); err != nil {
			return fmt.Errorf("failed to generate YAML docs: %v", err)
		}
		fmt.Printf("YAML documentation generated in %s/\n", yamlDir)

	case "rest":
		fmt.Println("Generating reStructuredText documentation...")
		if err := doc.GenReSTTree(rootCmd, restDir); err != nil {
			return fmt.Errorf("failed to generate ReST docs: %v", err)
		}
		fmt.Printf("reStructuredText documentation generated in %s/\n", restDir)

	case "all":
		fmt.Println("Generating all documentation formats...")

		// Generate man pages
		if err := doc.GenManTree(rootCmd, &doc.GenManHeader{
			Title:   "ABAPER",
			Section: "1",
			Manual:  "ABAPER Manual",
			Source:  fmt.Sprintf("ABAPER %s", Version),
		}, manDir); err != nil {
			return fmt.Errorf("failed to generate man pages: %v", err)
		}

		// Generate markdown
		if err := doc.GenMarkdownTree(rootCmd, mdDir); err != nil {
			return fmt.Errorf("failed to generate markdown docs: %v", err)
		}

		// Generate YAML
		if err := doc.GenYamlTree(rootCmd, yamlDir); err != nil {
			return fmt.Errorf("failed to generate YAML docs: %v", err)
		}

		// Generate ReST
		if err := doc.GenReSTTree(rootCmd, restDir); err != nil {
			return fmt.Errorf("failed to generate ReST docs: %v", err)
		}

		fmt.Printf("All documentation formats generated in %s/\n", outputDir)

	default:
		return fmt.Errorf("unknown documentation format: %s\nAvailable formats: man, markdown, yaml, rest, all", format)
	}

	return nil
}

func init() {
	// Add flags to docs command
	docsCmd.Flags().StringP("output", "o", "docs", "Output directory for generated documentation")

	// Add docs command to root command
	// rootCmd.AddCommand(docsCmd)
}
