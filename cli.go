package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bluefunda/abaper/types"
)

// CommandConfig holds command-specific configuration
type CommandConfig struct {
	Action     string // get, connect, search, list
	ObjectType string // program, class, function, etc.
	ObjectName string
	Args       []string // Additional arguments
}

// normalizeObjectType normalizes object type strings
func normalizeObjectType(objectType string) string {
	switch strings.ToUpper(objectType) {
	case "PROG", "REPORT":
		return "PROGRAM"
	case "CLAS":
		return "CLASS"
	case "FUNC", "FUGR":
		return "FUNCTION"
	case "INTF":
		return "INTERFACE"
	case "INCL":
		return "INCLUDE"
	case "TABL", "DDIC":
		return "TABLE"
	case "STRU", "TTYP":
		return "STRUCTURE"
	case "PACK", "DEVC":
		return "PACKAGE"
	default:
		return strings.ToUpper(objectType)
	}
}

// readSourceInput reads source code from various input sources
func readSourceInput(input string) (string, error) {
	// Handle special stdin case
	if input == "-" || input == "stdin" {
		content, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("failed to read from stdin: %w", err)
		}
		return string(content), nil
	}
	
	// Check if input looks like a file path
	if strings.Contains(input, "/") || strings.Contains(input, "\\") || 
	   filepath.Ext(input) != "" || len(input) > 200 {
		
		// Try to read as file
		if _, err := os.Stat(input); err == nil {
			content, err := os.ReadFile(input)
			if err != nil {
				return "", fmt.Errorf("failed to read file %s: %w", input, err)
			}
			return string(content), nil
		}
	}
	
	// Return error if we can't read it as a file
	return "", fmt.Errorf("not a valid file path or file doesn't exist: %s", input)
}

// HandleGet retrieves ABAP object source code
func HandleGet(config *CommandConfig, adtClient types.ADTClient, quiet bool, normal bool) error {
	if config.ObjectType == "" {
		return fmt.Errorf("object type required for get action")
	}
	if config.ObjectName == "" {
		return fmt.Errorf("object name required for get action")
	}

	objectType := normalizeObjectType(config.ObjectType)
	objectName := strings.ToUpper(config.ObjectName)

	if !quiet || normal {
		fmt.Printf("📄 Retrieving %s %s...\n", objectType, objectName)
	}

	var source *types.ADTSourceCode
	var err error

	switch objectType {
	case "PROGRAM":
		source, err = adtClient.GetProgram(objectName)
	case "CLASS":
		source, err = adtClient.GetClass(objectName)
	case "FUNCTION":
		if len(config.Args) == 0 {
			return fmt.Errorf("function group required for function: %s get function <n> <group>", "abaper")
		}
		functionGroup := strings.ToUpper(config.Args[0])
		source, err = adtClient.GetFunction(objectName, functionGroup)
	case "INCLUDE":
		source, err = adtClient.GetInclude(objectName)
	case "INTERFACE":
		source, err = adtClient.GetInterface(objectName)
	case "STRUCTURE":
		source, err = adtClient.GetStructure(objectName)
	case "TABLE":
		source, err = adtClient.GetTable(objectName)
	case "PACKAGE":
		return HandleGetPackage(config, adtClient, quiet, normal)
	default:
		return fmt.Errorf("unsupported object type: %s", objectType)
	}

	if err != nil {
		return fmt.Errorf("failed to retrieve %s %s: %w", objectType, objectName, err)
	}

	// Always output the source code (even in quiet mode, this is the primary output)
	fmt.Printf("\n=== %s %s ===\n", objectType, objectName)
	fmt.Printf("Type: %s\n", source.ObjectType)
	if source.Version != "" {
		fmt.Printf("Version: %s\n", source.Version)
	}
	fmt.Printf("\nSource Code:\n")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println(source.Source)
	fmt.Println(strings.Repeat("=", 80))

	return nil
}

// HandleGetPackage retrieves package contents
func HandleGetPackage(config *CommandConfig, adtClient types.ADTClient, quiet bool, normal bool) error {
	packageName := strings.ToUpper(config.ObjectName)

	if !quiet || normal {
		fmt.Printf("📦 Retrieving package %s...\n", packageName)
	}

	packageInfo, err := adtClient.GetPackageContents(packageName)
	if err != nil {
		return fmt.Errorf("failed to get package: %w", err)
	}

	fmt.Printf("\n=== Package %s ===\n", packageName)
	fmt.Printf("Name: %s\n", packageInfo.Name)
	fmt.Printf("Description: %s\n", packageInfo.Description)
	fmt.Printf("Objects: %d\n", len(packageInfo.Objects))

	if len(packageInfo.Objects) > 0 {
		fmt.Printf("\nObjects in Package:\n")
		fmt.Println(strings.Repeat("=", 80))

		// Group objects by type
		objectsByType := make(map[string][]types.ADTObject)
		for _, obj := range packageInfo.Objects {
			objectsByType[obj.Type] = append(objectsByType[obj.Type], obj)
		}

		for objectType, objects := range objectsByType {
			fmt.Printf("\n%s (%d objects):\n", objectType, len(objects))
			for _, obj := range objects {
				fmt.Printf("  • %s", obj.Name)
				if obj.Description != "" {
					fmt.Printf(" - %s", obj.Description)
				}
				fmt.Println()
			}
		}
		fmt.Println(strings.Repeat("=", 80))
	}

	return nil
}

// HandleSearch searches for ABAP objects
func HandleSearch(config *CommandConfig, adtClient types.ADTClient, quiet bool, normal bool) error {
	if config.ObjectType != "objects" {
		return fmt.Errorf("search type must be 'objects': %s search objects <pattern>", "abaper")
	}
	if config.ObjectName == "" {
		return fmt.Errorf("search pattern required")
	}

	pattern := config.ObjectName
	var objectTypes []string
	for _, arg := range config.Args {
		objectTypes = append(objectTypes, normalizeObjectType(arg))
	}

	if !quiet || normal {
		fmt.Printf("🔍 Searching for '%s'", pattern)
		if len(objectTypes) > 0 {
			fmt.Printf(" (types: %s)", strings.Join(objectTypes, ", "))
		}
		fmt.Println("...")
	}

	results, err := adtClient.SearchObjects(pattern, objectTypes)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	fmt.Printf("\n=== Search Results ===\n")
	fmt.Printf("Pattern: %s\n", pattern)
	fmt.Printf("Total Results: %d\n", results.Total)

	if len(results.Objects) > 0 {
		fmt.Printf("\nFound Objects:\n")
		fmt.Println(strings.Repeat("=", 80))

		// Group results by type
		resultsByType := make(map[string][]types.ADTObject)
		for _, obj := range results.Objects {
			resultsByType[obj.Type] = append(resultsByType[obj.Type], obj)
		}

		for objectType, objects := range resultsByType {
			fmt.Printf("\n%s (%d objects):\n", objectType, len(objects))
			for _, obj := range objects {
				fmt.Printf("  • %s", obj.Name)
				if obj.Description != "" {
					fmt.Printf(" - %s", obj.Description)
				}
				if obj.Package != "" {
					fmt.Printf(" (Package: %s)", obj.Package)
				}
				fmt.Println()
			}
		}
		fmt.Println(strings.Repeat("=", 80))
	} else {
		fmt.Printf("\nNo objects found matching pattern '%s'.\n", pattern)
	}

	return nil
}

// HandleList lists objects (packages, etc.)
func HandleList(config *CommandConfig, adtClient types.ADTClient, quiet bool, normal bool) error {
	if config.ObjectType == "" {
		return fmt.Errorf("list type required: %s list packages [pattern]", "abaper")
	}

	listType := strings.ToLower(config.ObjectType)

	switch listType {
	case "packages", "package":
		return HandleListPackages(config, adtClient, quiet, normal)
	default:
		return fmt.Errorf("unsupported list type: %s", listType)
	}
}

// HandleListPackages lists packages
func HandleListPackages(config *CommandConfig, adtClient types.ADTClient, quiet bool, normal bool) error {
	pattern := config.ObjectName
	if pattern == "" {
		pattern = "*"
	}

	if !quiet || normal {
		fmt.Printf("📦 Listing packages matching '%s'...\n", pattern)
	}

	packages, err := adtClient.ListPackages(pattern)
	if err != nil {
		return fmt.Errorf("failed to list packages: %w", err)
	}

	fmt.Printf("\n=== Packages ===\n")
	fmt.Printf("Found %d packages:\n", len(packages))
	fmt.Println(strings.Repeat("=", 50))

	for _, pkg := range packages {
		fmt.Printf("• %s", pkg.Name)
		if pkg.Description != "" {
			fmt.Printf(" - %s", pkg.Description)
		}
		fmt.Println()
	}

	return nil
}

// HandleConnect tests ADT connection
func HandleConnect(config *CommandConfig, adtClient types.ADTClient, quiet bool, normal bool) error {
	if !quiet || normal {
		fmt.Println("🔌 Testing ADT connection...")
	}

	if adtClient == nil {
		return fmt.Errorf("ADT client not configured")
	}

	if err := adtClient.TestConnection(); err != nil {
		fmt.Printf("❌ Connection failed: %v\n", err)
		fmt.Println("\n💡 Troubleshooting tips:")
		fmt.Println("  1. Check your SAP credentials and host configuration")
		fmt.Println("  2. Verify ADT services are activated in SICF (transaction SICF)")
		fmt.Println("  3. Ensure your user has S_DEVELOP authorization")
		fmt.Println("  4. Test basic connectivity: ping <host>")
		return err
	}

	fmt.Println("✅ ADT connection successful!")
	return nil
}

// getObjectSource retrieves source code for any supported object type
func getObjectSource(config *CommandConfig, adtClient types.ADTClient) (*types.ADTSourceCode, error) {
	objectType := normalizeObjectType(config.ObjectType)
	objectName := strings.ToUpper(config.ObjectName)

	switch objectType {
	case "PROGRAM":
		return adtClient.GetProgram(objectName)
	case "CLASS":
		return adtClient.GetClass(objectName)
	case "FUNCTION":
		if len(config.Args) == 0 {
			return nil, fmt.Errorf("function group required for function")
		}
		functionGroup := strings.ToUpper(config.Args[0])
		return adtClient.GetFunction(objectName, functionGroup)
	case "INCLUDE":
		return adtClient.GetInclude(objectName)
	case "INTERFACE":
		return adtClient.GetInterface(objectName)
	case "STRUCTURE":
		return adtClient.GetStructure(objectName)
	case "TABLE":
		return adtClient.GetTable(objectName)
	default:
		return nil, fmt.Errorf("unsupported object type for source retrieval: %s", objectType)
	}
}

// CreateADTClient creates ADT client from configuration
func CreateADTClient(config *Config) (types.ADTClient, error) {
	if config.ADTHost == "" {
		return nil, fmt.Errorf("ADT host not configured (use --adt-host or set SAP_HOST)")
	}

	if config.ADTUsername == "" {
		return nil, fmt.Errorf("ADT username not configured (use --adt-username or set SAP_USERNAME)")
	}

	if config.ADTPassword == "" {
		return nil, fmt.Errorf("ADT password not configured (use --adt-password or set SAP_PASSWORD)")
	}

	adtConfig := &types.ADTConfig{
		Host:     config.ADTHost,
		Client:   config.ADTClient,
		Username: config.ADTUsername,
		Password: config.ADTPassword,
		Language: "EN",
		// Enable SSL handling for port 50000
		AllowSelfSigned: true,
		ConnectTimeout:  30,
		RequestTimeout:  120,
		Debug:           false,
	}

	// Set default client if not specified
	if adtConfig.Client == "" {
		adtConfig.Client = "100"
	}

	client := NewADTClient(adtConfig)

	// Force stateful session BEFORE authentication
	client.SetSessionType(types.SessionStateful)

	// Test authentication
	if err := client.Authenticate(); err != nil {
		return nil, fmt.Errorf("ADT authentication failed: %w", err)
	}

	return client, nil
}

// HandleCreate creates new ABAP objects with enhanced functionality
func HandleCreate(config *CommandConfig, adtClient types.ADTClient, quiet bool, normal bool) error {
	if config.ObjectType == "" {
		return fmt.Errorf("object type required for create action")
	}
	if config.ObjectName == "" {
		return fmt.Errorf("object name required for create action")
	}

	objectType := normalizeObjectType(config.ObjectType)
	objectName := strings.ToUpper(config.ObjectName)

	// Parse additional arguments for creation options
	description := fmt.Sprintf("%s %s", objectType, objectName)
	packageName := "$TMP" // Default to local package
	sourceCode := ""

	// Parse arguments: description, package, source_file_or_content
	if len(config.Args) > 0 {
		description = config.Args[0]
	}
	if len(config.Args) > 1 {
		packageName = strings.ToUpper(config.Args[1])
	}
	if len(config.Args) > 2 {
		// Third argument could be source code or file path
		sourceInput := config.Args[2]
		
		// Check if it's a file path by trying to read it
		if sourceInput != "" {
			if fileContent, err := readSourceInput(sourceInput); err == nil {
				sourceCode = fileContent
				if !quiet || normal {
					fmt.Printf("   Source file: %s (%d characters)\n", sourceInput, len(sourceCode))
				}
			} else {
				// If file reading fails, treat it as direct source code content
				sourceCode = sourceInput
			}
		}
	}

	if !quiet || normal {
		fmt.Printf("📝 Creating %s %s...\n", objectType, objectName)
		fmt.Printf("   Description: %s\n", description)
		fmt.Printf("   Package: %s\n", packageName)
		if sourceCode != "" {
			fmt.Printf("   Source: %d characters\n", len(sourceCode))
		}
	}

	var err error

	switch objectType {
	case "PROGRAM":
		if sourceCode == "" {
			// Generate basic program template
			sourceCode = fmt.Sprintf("REPORT %s.\n\nWRITE: 'Hello from %s!'.\n\nSTART-OF-SELECTION.\n  WRITE: / 'Program %s created successfully.'.\n  WRITE: / 'Current date:', sy-datum.\n  WRITE: / 'Current time:', sy-uzeit.\n", objectName, objectName, objectName)
		}
		err = adtClient.CreateProgram(objectName, description, sourceCode)
	case "CLASS":
		if sourceCode == "" {
			// Generate basic class template
			sourceCode = fmt.Sprintf("CLASS %s DEFINITION PUBLIC FINAL CREATE PUBLIC.\n  PUBLIC SECTION.\n    METHODS: say_hello RETURNING VALUE(rv_message) TYPE string.\nENDCLASS.\n\nCLASS %s IMPLEMENTATION.\n  METHOD say_hello.\n    rv_message = 'Hello from %s!'.\n  ENDMETHOD.\nENDCLASS.\n", objectName, objectName, objectName)
		}
		err = adtClient.CreateClass(objectName, description, sourceCode)
	case "INCLUDE":
		if sourceCode == "" {
			sourceCode = fmt.Sprintf("*&---------------------------------------------------------------------*\n*& Include %s\n*&---------------------------------------------------------------------*\n\n* Include %s created by abaper CLI\n", objectName, objectName)
		}
		err = adtClient.CreateInclude(objectName, description, sourceCode)
	case "INTERFACE":
		if sourceCode == "" {
			sourceCode = fmt.Sprintf("INTERFACE %s PUBLIC.\n  METHODS: do_something.\nENDINTERFACE.\n", objectName)
		}
		err = adtClient.CreateInterface(objectName, description, sourceCode)
	case "STRUCTURE":
		err = adtClient.CreateStructure(objectName, description, sourceCode)
	case "TABLE":
		err = adtClient.CreateTable(objectName, description, sourceCode)
	case "FUNCTIONGROUP":
		err = adtClient.CreateFunctionGroup(objectName, description, sourceCode)
	default:
		return fmt.Errorf("unsupported object type for creation: %s", objectType)
	}

	if err != nil {
		return fmt.Errorf("failed to create %s %s: %w", objectType, objectName, err)
	}

	if !quiet {
		fmt.Printf("✅ %s %s created successfully!\n", objectType, objectName)
		if packageName != "$TMP" {
			fmt.Printf("   Package: %s\n", packageName)
		}
		if sourceCode != "" {
			fmt.Printf("   Source code inserted and activated\n")
		}
	}

	return nil
}

// HandleUpdate updates ABAP object source code
func HandleUpdate(config *CommandConfig, adtClient types.ADTClient, quiet bool, normal bool) error {
	if config.ObjectType == "" {
		return fmt.Errorf("object type required for update action")
	}
	if config.ObjectName == "" {
		return fmt.Errorf("object name required for update action")
	}
	if len(config.Args) == 0 {
		return fmt.Errorf("source file/input required for update action")
	}

	objectType := normalizeObjectType(config.ObjectType)
	objectName := strings.ToUpper(config.ObjectName)
	sourceInput := config.Args[0] // First argument is source file/input

	if !quiet || normal {
		fmt.Printf("🔄 Updating %s %s...\n", objectType, objectName)
	}

	// Read source code from input (file, stdin, or direct)
	sourceCode, err := readSourceInput(sourceInput)
	if err != nil {
		// If file reading fails, treat it as direct source code content
		sourceCode = sourceInput
		if !quiet || normal {
			fmt.Printf("   Source: %d characters (direct input)\n", len(sourceCode))
		}
	} else {
		if !quiet || normal {
			if sourceInput == "-" || sourceInput == "stdin" {
				fmt.Printf("   Source file: stdin (%d characters)\n", len(sourceCode))
			} else {
				fmt.Printf("   Source file: %s (%d characters)\n", sourceInput, len(sourceCode))
			}
		}
	}

	// Validate source code
	if strings.TrimSpace(sourceCode) == "" {
		return fmt.Errorf("source code cannot be empty")
	}

	// Update the object based on type
	switch objectType {
	case "PROGRAM":
		err = adtClient.UpdateProgram(objectName, sourceCode)
	case "CLASS":
		err = adtClient.UpdateClass(objectName, sourceCode)
	case "INCLUDE":
		err = adtClient.UpdateInclude(objectName, sourceCode)
	case "INTERFACE":
		err = adtClient.UpdateInterface(objectName, sourceCode)
	default:
		return fmt.Errorf("unsupported object type for update: %s", objectType)
	}

	if err != nil {
		return fmt.Errorf("failed to update %s %s: %w", objectType, objectName, err)
	}

	if !quiet {
		fmt.Printf("✅ %s %s updated successfully!\n", objectType, objectName)
		fmt.Printf("   Source code updated and activated\n")
	}

	return nil
}
