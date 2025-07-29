package main

import (
	"fmt"
	"strings"
)

// POSIX-compliant command execution
// This file replaces the complex cli_commands.go with a simpler, POSIX-compliant approach

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

// handleGet retrieves ABAP object source code
func handleGet(config *Config, adtClient *ADTClient) error {
	if config.ObjectType == "" {
		return fmt.Errorf("object type required for get action")
	}
	if config.ObjectName == "" {
		return fmt.Errorf("object name required for get action")
	}

	objectType := normalizeObjectType(config.ObjectType)
	objectName := strings.ToUpper(config.ObjectName)

	fmt.Printf("📄 Retrieving %s %s...\n", objectType, objectName)

	var source *ADTSourceCode
	var err error

	switch objectType {
	case "PROGRAM":
		source, err = adtClient.GetProgram(objectName)
	case "CLASS":
		source, err = adtClient.GetClass(objectName)
	case "FUNCTION":
		if len(config.Args) == 0 {
			return fmt.Errorf("function group required for function: %s get function <n> <group>", PROGRAM_NAME)
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
		return handleGetPackage(config, adtClient)
	default:
		return fmt.Errorf("unsupported object type: %s", objectType)
	}

	if err != nil {
		return fmt.Errorf("failed to retrieve %s %s: %w", objectType, objectName, err)
	}

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

// handleGetPackage retrieves package contents
func handleGetPackage(config *Config, adtClient *ADTClient) error {
	packageName := strings.ToUpper(config.ObjectName)

	fmt.Printf("📦 Retrieving package %s...\n", packageName)

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
		objectsByType := make(map[string][]ADTObject)
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

// handleSearch searches for ABAP objects
func handleSearch(config *Config, adtClient *ADTClient) error {
	if config.ObjectType != "objects" {
		return fmt.Errorf("search type must be 'objects': %s search objects <pattern>", PROGRAM_NAME)
	}
	if config.ObjectName == "" {
		return fmt.Errorf("search pattern required")
	}

	pattern := config.ObjectName
	var objectTypes []string
	for _, arg := range config.Args {
		objectTypes = append(objectTypes, normalizeObjectType(arg))
	}

	fmt.Printf("🔍 Searching for '%s'", pattern)
	if len(objectTypes) > 0 {
		fmt.Printf(" (types: %s)", strings.Join(objectTypes, ", "))
	}
	fmt.Println("...")

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
		resultsByType := make(map[string][]ADTObject)
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

// handleList lists objects (packages, etc.)
func handleList(config *Config, adtClient *ADTClient) error {
	if config.ObjectType == "" {
		return fmt.Errorf("list type required: %s list packages [pattern]", PROGRAM_NAME)
	}

	listType := strings.ToLower(config.ObjectType)

	switch listType {
	case "packages", "package":
		return handleListPackages(config, adtClient)
	default:
		return fmt.Errorf("unsupported list type: %s", listType)
	}
}

// handleListPackages lists packages
func handleListPackages(config *Config, adtClient *ADTClient) error {
	pattern := config.ObjectName
	if pattern == "" {
		pattern = "*"
	}

	fmt.Printf("📦 Listing packages matching '%s'...\n", pattern)

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

// handleConnect tests ADT connection
func handleConnect(config *Config, adtClient *ADTClient) error {
	fmt.Println("🔌 Testing ADT connection...")

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

// handleHelp shows help information
func handleHelp(config *Config) error {
	if config.ObjectType != "" {
		// Show specific command help
		return showCommandHelp(config.ObjectType)
	}

	printHelp()
	return nil
}

// showCommandHelp shows help for specific commands
func showCommandHelp(command string) error {
	switch command {
	case "get":
		fmt.Printf(`Usage: %s get TYPE NAME [ARGS...]

Retrieve ABAP object source code.

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
  %s get program ZTEST
  %s get class ZCL_TEST
  %s get function ZTEST_FUNC ZTEST_GROUP
  %s get package $TMP
`, PROGRAM_NAME, PROGRAM_NAME, PROGRAM_NAME, PROGRAM_NAME, PROGRAM_NAME)

	case "search":
		fmt.Printf(`Usage: %s search objects PATTERN [TYPES...]

Search for ABAP objects by pattern.

EXAMPLES:
  %s search objects "Z*"
  %s search objects "CL_*" class
  %s search objects "*TEST*" program class
`, PROGRAM_NAME, PROGRAM_NAME, PROGRAM_NAME, PROGRAM_NAME)

	case "list":
		fmt.Printf(`Usage: %s list TYPE [PATTERN]

List objects of specified type.

TYPES:
  packages    List packages

EXAMPLES:
  %s list packages
  %s list packages "Z*"
`, PROGRAM_NAME, PROGRAM_NAME, PROGRAM_NAME)

	case "connect":
		fmt.Printf(`Usage: %s connect

Test ADT connection to SAP system.

This command verifies:
- Basic connectivity to SAP system
- ADT service availability
- Authentication credentials
- User permissions

EXAMPLES:
  %s connect
`, PROGRAM_NAME, PROGRAM_NAME)

	default:
		return fmt.Errorf("unknown command: %s", command)
	}

	return nil
}

// Helper functions

// getObjectSource retrieves source code for any supported object type
func getObjectSource(config *Config, adtClient *ADTClient) (*ADTSourceCode, error) {
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

// Create ADT client from configuration
func createADTClient(config *Config) (*ADTClient, error) {
	if config.ADTHost == "" {
		return nil, fmt.Errorf("ADT host not configured (use --adt-host or set SAP_HOST)")
	}

	if config.ADTUsername == "" {
		return nil, fmt.Errorf("ADT username not configured (use --adt-username or set SAP_USERNAME)")
	}

	if config.ADTPassword == "" {
		return nil, fmt.Errorf("ADT password not configured (use --adt-password or set SAP_PASSWORD)")
	}

	adtConfig := &ADTConfig{
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
	client.SetSessionType(SessionStateful)

	// Test authentication
	if err := client.Authenticate(); err != nil {
		return nil, fmt.Errorf("ADT authentication failed: %w", err)
	}

	return client, nil
}
