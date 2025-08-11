package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/bluefunda/abaper/rest/server"
	"github.com/bluefunda/abaper/types"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// POSIX-compliant exit codes
const (
	ExitSuccess      = 0   // Successful completion
	ExitGeneralError = 1   // General error
	ExitMisuse       = 2   // Misuse of shell command
	ExitSIGINT       = 130 // Terminated by Ctrl+C (128 + 2)
	ExitSIGTERM      = 143 // Terminated by SIGTERM (128 + 15)
)

// Build-time variables set via ldflags
var (
	Version   string = "v0.0.3"
	BuildTime string = "unknown"
	GitCommit string = "unknown"
	BuildMode string = "dev"
)

// Configuration
type Config struct {
	// Mode
	Mode string // "server", "cli"
	Port string

	// Common flags - QUIET IS NOW DEFAULT
	Quiet   bool // Default: true
	Verbose bool
	Normal  bool

	// ADT Configuration
	ADTHost     string
	ADTClient   string
	ADTUsername string
	ADTPassword string

	// File logging support
	LogFile    string
	ConfigFile string
}

const (
	PROGRAM_NAME = "abaper"
)

var (
	logger     *zap.Logger
	rootConfig = &Config{}

	// Global ADT client cache for connection reuse - now uses shared interface
	cachedADTClient types.ADTClient
	cachedADTConfig string
	cacheTime       time.Time
	cacheTimeout    = 30 * time.Minute
)

// Root command
var rootCmd = &cobra.Command{
	Use:   PROGRAM_NAME,
	Short: "ABAP Development Tool - CLI and REST Services (No AI)",
	Long: `ABAP Development Tool - CLI and REST Services (No AI)

A comprehensive CLI tool for interacting with SAP ABAP systems via ADT.
Supports retrieving source code, searching objects, and testing connections.`,
	Version: Version,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize logger
		initLogger(rootConfig.Verbose, rootConfig.Quiet && !rootConfig.Normal, rootConfig.LogFile)

		// Setup signal handling
		setupSignalHandling()

		// Setup ADT cache cleanup on exit
		go func() {
			defer cleanupADTCache()
		}()

		// Log startup
		logger.Info("Application starting",
			zap.String("version", Version),
			zap.String("build_mode", BuildMode),
			zap.String("build_time", BuildTime),
			zap.String("git_commit", GitCommit),
			zap.String("mode", rootConfig.Mode))

		return nil
	},
}

// Server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Run as REST API server",
	Long:  "Start the ABAPER REST API server for HTTP-based ABAP operations.",
	RunE: func(cmd *cobra.Command, args []string) error {
		rootConfig.Mode = "server"
		return runServerMode(rootConfig)
	},
}

// Get command
var getCmd = &cobra.Command{
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
	RunE: func(cmd *cobra.Command, args []string) error {
		rootConfig.Mode = "cli"

		config := &CommandConfig{
			Action:     "get",
			ObjectType: args[0],
			ObjectName: args[1],
			Args:       args[2:],
		}

		adtClient, err := getCachedADTClient(rootConfig)
		if err != nil {
			return fmt.Errorf("failed to create ADT client: %w", err)
		}

		return HandleGet(config, adtClient, rootConfig.Quiet, rootConfig.Normal)
	},
}

// Search command
var searchCmd = &cobra.Command{
	Use:   "search objects PATTERN [TYPES...]",
	Short: "Search for ABAP objects",
	Long: `Search for ABAP objects by pattern.

EXAMPLES:
  abaper search objects "Z*"
  abaper search objects "CL_*" class
  abaper search objects "*TEST*" program class`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		rootConfig.Mode = "cli"

		if args[0] != "objects" {
			return fmt.Errorf("search type must be 'objects'")
		}

		config := &CommandConfig{
			Action:     "search",
			ObjectType: args[0],
			ObjectName: args[1],
			Args:       args[2:],
		}

		adtClient, err := getCachedADTClient(rootConfig)
		if err != nil {
			return fmt.Errorf("failed to create ADT client: %w", err)
		}

		return HandleSearch(config, adtClient, rootConfig.Quiet, rootConfig.Normal)
	},
}

// List command
var listCmd = &cobra.Command{
	Use:   "list TYPE [PATTERN]",
	Short: "List objects of specified type",
	Long: `List objects of specified type.

TYPES:
  packages    List packages

EXAMPLES:
  abaper list packages
  abaper list packages "Z*"`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		rootConfig.Mode = "cli"

		config := &CommandConfig{
			Action:     "list",
			ObjectType: args[0],
		}

		if len(args) > 1 {
			config.ObjectName = args[1]
		}

		adtClient, err := getCachedADTClient(rootConfig)
		if err != nil {
			return fmt.Errorf("failed to create ADT client: %w", err)
		}

		return HandleList(config, adtClient, rootConfig.Quiet, rootConfig.Normal)
	},
}

// Connect command
var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Test ADT connection",
	Long: `Test ADT connection to SAP system.

This command verifies:
- Basic connectivity to SAP system
- ADT service availability
- Authentication credentials
- User permissions`,
	RunE: func(cmd *cobra.Command, args []string) error {
		rootConfig.Mode = "cli"

		config := &CommandConfig{
			Action: "connect",
		}

		adtClient, err := getCachedADTClient(rootConfig)
		if err != nil {
			if config.Action == "connect" {
				// For connect command, show the error but continue to demonstrate the problem
				fmt.Printf("❌ ADT connection failed: %v\n", err)
				return err
			}
			return fmt.Errorf("failed to create ADT client: %w", err)
		}

		return HandleConnect(config, adtClient, rootConfig.Quiet, rootConfig.Normal)
	},
}

// Update command
var updateCmd = &cobra.Command{
	Use:   "update TYPE NAME SOURCE",
	Short: "Update ABAP object with new source code",
	Long: `Update ABAP object in SAP system with new source code.

The program must already exist. If it doesn't exist, an error will be thrown.

TYPES:
  program     ABAP program/report
  class       ABAP class
  include     ABAP include
  interface   ABAP interface

SOURCE can be:
  - File path (e.g., ./myprogram.abap)
  - "-" or "stdin" to read from standard input

EXAMPLES:
  abaper update program ZTEST ./source.abap
  abaper update program ZTEST -
  echo 'REPORT ZTEST.' | abaper update program ZTEST stdin`,
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		rootConfig.Mode = "cli"

		config := &CommandConfig{
			Action:     "update",
			ObjectType: args[0],
			ObjectName: args[1],
			Args:       []string{args[2]}, // source file/input
		}

		adtClient, err := getCachedADTClient(rootConfig)
		if err != nil {
			return fmt.Errorf("failed to create ADT client: %w", err)
		}

		return HandleUpdate(config, adtClient, rootConfig.Quiet, rootConfig.Normal)
	},
}

// Create command
var createCmd = &cobra.Command{
	Use:   "create TYPE NAME [DESCRIPTION] [PACKAGE] [SOURCE]",
	Short: "Create ABAP object with optional source code",
	Long: `Create ABAP object in SAP system with optional source code.

TYPES:
  program     ABAP program/report
  class       ABAP class
  function    ABAP function module (requires function group)
  include     ABAP include
  interface   ABAP interface
  structure   ABAP structure
  table       ABAP table
  package     ABAP package contents

SOURCE can be:
  - File path (e.g., ./myprogram.abap)
  - "-" or "stdin" to read from standard input
  - Direct ABAP source code (if not a valid file)

EXAMPLES:
  abaper create program ZTEST
  abaper create program ZTEST "My test program" 
  abaper create program ZTEST "My test program" $TMP
  abaper create program ZTEST "My test program" $TMP ./source.abap
  abaper create program ZTEST "My test program" $TMP -
  echo 'REPORT ZTEST.' | abaper create program ZTEST "My test" $TMP stdin`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		rootConfig.Mode = "cli"

		config := &CommandConfig{
			Action:     "create",
			ObjectType: args[0],
			ObjectName: args[1],
			Args:       args[2:],
		}

		adtClient, err := getCachedADTClient(rootConfig)
		if err != nil {
			return fmt.Errorf("failed to create ADT client: %w", err)
		}

		return HandleCreate(config, adtClient, rootConfig.Quiet, rootConfig.Normal)
	},
}

func set_host() string {
	sapHost := os.Getenv("SAP_HOST")
	sapPort := os.Getenv("SAP_PORT")
	if sapHost != "" && sapPort != "" {
		return sapHost + ":" + sapPort
	} else {
		return os.Getenv("SAP_HOST") // fallback to original behavior
	}
}

// Enhanced logger with file support and quiet mode default
func initLogger(verbose, quiet bool, logFile string) {
	var outputPaths, errorPaths []string

	// Configure output based on flags and log file
	if logFile != "" {
		// Create log directory if needed
		logDir := filepath.Dir(logFile)
		if logDir != "." && logDir != "" {
			os.MkdirAll(logDir, 0755)
		}

		outputPaths = []string{logFile}
		errorPaths = []string{logFile}

		// In verbose mode, also output to stderr
		if verbose {
			outputPaths = append(outputPaths, "stderr")
			errorPaths = append(errorPaths, "stderr")
		}
	} else if !quiet {
		// Only output to stderr if not in quiet mode
		outputPaths = []string{"stderr"}
		errorPaths = []string{"stderr"}
	} else {
		// Quiet mode: only errors to stderr
		outputPaths = []string{}
		errorPaths = []string{"stderr"}
	}

	var config zap.Config

	if verbose {
		// Verbose mode
		config = zap.Config{
			Level:       zap.NewAtomicLevelAt(zap.DebugLevel),
			Development: true,
			Encoding:    "console",
			EncoderConfig: zapcore.EncoderConfig{
				TimeKey:        "T",
				LevelKey:       "L",
				NameKey:        "N",
				CallerKey:      "C",
				MessageKey:     "M",
				StacktraceKey:  "S",
				LineEnding:     zapcore.DefaultLineEnding,
				EncodeLevel:    zapcore.CapitalColorLevelEncoder,
				EncodeTime:     zapcore.TimeEncoderOfLayout("15:04:05"),
				EncodeDuration: zapcore.StringDurationEncoder,
				EncodeCaller:   zapcore.ShortCallerEncoder,
			},
			OutputPaths:      outputPaths,
			ErrorOutputPaths: errorPaths,
		}
	} else if quiet {
		// Quiet mode: minimal logging
		config = zap.Config{
			Level:            zap.NewAtomicLevelAt(zap.WarnLevel),
			Development:      false,
			Encoding:         "json",
			OutputPaths:      outputPaths,
			ErrorOutputPaths: errorPaths,
			EncoderConfig: zapcore.EncoderConfig{
				TimeKey:     "timestamp",
				LevelKey:    "level",
				MessageKey:  "message",
				LineEnding:  zapcore.DefaultLineEnding,
				EncodeLevel: zapcore.LowercaseLevelEncoder,
				EncodeTime:  zapcore.ISO8601TimeEncoder,
			},
		}
	} else {
		// Normal mode
		config = zap.Config{
			Level:       zap.NewAtomicLevelAt(zap.InfoLevel),
			Development: false,
			Encoding:    "json",
			EncoderConfig: zapcore.EncoderConfig{
				TimeKey:        "ts",
				LevelKey:       "level",
				NameKey:        "logger",
				CallerKey:      "caller",
				MessageKey:     "msg",
				StacktraceKey:  "stacktrace",
				LineEnding:     zapcore.DefaultLineEnding,
				EncodeLevel:    zapcore.LowercaseLevelEncoder,
				EncodeTime:     zapcore.EpochTimeEncoder,
				EncodeDuration: zapcore.SecondsDurationEncoder,
				EncodeCaller:   zapcore.ShortCallerEncoder,
			},
			OutputPaths:      outputPaths,
			ErrorOutputPaths: errorPaths,
		}
	}

	var err error
	logger, err = config.Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to initialize logger: %v\n", PROGRAM_NAME, err)
		os.Exit(ExitGeneralError)
	}

	// Only show log file info if not in quiet mode
	if logFile != "" && !quiet {
		fmt.Printf("📄 Logging to: %s\n", logFile)
	}
}

// Signal handling
func setupSignalHandling() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.Info("Received signal, shutting down gracefully", zap.String("signal", sig.String()))

		logger.Sync()

		switch sig {
		case os.Interrupt:
			os.Exit(ExitSIGINT)
		case syscall.SIGTERM:
			os.Exit(ExitSIGTERM)
		default:
			os.Exit(ExitGeneralError)
		}
	}()
}

// getCachedADTClient returns cached client if valid, creates new one otherwise
func getCachedADTClient(config *Config) (types.ADTClient, error) {
	// Create cache key from config
	configKey := fmt.Sprintf("%s|%s|%s|%s",
		config.ADTHost, config.ADTClient, config.ADTUsername, config.ADTPassword)

	// Check if we have a valid cached client
	if cachedADTClient != nil &&
		cachedADTConfig == configKey &&
		time.Since(cacheTime) < cacheTimeout &&
		cachedADTClient.IsAuthenticated() {

		// Optional: Test connection with lightweight ping (can be disabled for performance)
		if err := cachedADTClient.TestConnection(); err != nil {
			logger.Info("Cached ADT client failed ping test, creating new client", zap.Error(err))
			// Continue to create new client
		} else {
			logger.Debug("Using cached ADT client",
				zap.String("host", config.ADTHost),
				zap.Duration("cache_age", time.Since(cacheTime)))
			return cachedADTClient, nil
		}
	}

	// Cache miss or expired - create new client
	if cachedADTClient != nil {
		logger.Info("ADT cache miss",
			zap.Bool("cache_expired", time.Since(cacheTime) >= cacheTimeout),
			zap.Bool("config_changed", cachedADTConfig != configKey),
			zap.Bool("not_authenticated", !cachedADTClient.IsAuthenticated()))
	} else {
		logger.Info("Creating first ADT client", zap.String("host", config.ADTHost))
	}

	client, err := CreateADTClient(config)
	if err != nil {
		return nil, err
	}

	// Cache the new client
	cachedADTClient = client
	cachedADTConfig = configKey
	cacheTime = time.Now()

	logger.Info("ADT client cached successfully",
		zap.String("host", config.ADTHost),
		zap.Duration("cache_timeout", cacheTimeout))

	return client, nil
}

// cleanupADTCache cleans up the ADT cache
func cleanupADTCache() {
	if cachedADTClient != nil {
		logger.Debug("Cleaning up ADT cache")
		cachedADTClient = nil
		cachedADTConfig = ""
		cacheTime = time.Time{}
	}
}

// runServerMode starts the REST server with CLI and ADT integration
func runServerMode(config *Config) error {
	logger.Info("Starting in server mode", zap.String("port", config.Port))

	// Create ADT client for server mode
	adtClient, err := getCachedADTClient(config)
	if err != nil {
		logger.Error("Failed to create ADT client for server mode", zap.Error(err))
		return fmt.Errorf("failed to create ADT client for server: %w", err)
	}

	logger.Info("ADT client created successfully for server mode",
		zap.String("host", config.ADTHost),
		zap.Bool("authenticated", adtClient.IsAuthenticated()))

	serverConfig := &server.Config{
		ADTHost:     config.ADTHost,
		ADTClient:   config.ADTClient,
		ADTUsername: config.ADTUsername,
		ADTPassword: config.ADTPassword,
		Verbose:     config.Verbose,
		Quiet:       config.Quiet && !config.Normal,
	}

	// Pass ADT client directly to server - no adapter needed!
	restServer := server.NewRestServer(serverConfig, logger, adtClient)
	restServer.Start(config.Port)
	return nil
}

// Error handling helper
func exitWithError(err error, exitCode int) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", PROGRAM_NAME, err)
	if logger != nil {
		logger.Error("Application error", zap.Error(err), zap.Int("exit_code", exitCode))
		logger.Sync()
	}
	os.Exit(exitCode)
}

// Execute POSIX-style command
func executeCommand(config *CommandConfig, adtClient types.ADTClient) error {
	switch config.Action {
	case "get":
		return HandleGet(config, adtClient, rootConfig.Quiet, rootConfig.Normal)
	case "search":
		return HandleSearch(config, adtClient, rootConfig.Quiet, rootConfig.Normal)
	case "list":
		return HandleList(config, adtClient, rootConfig.Quiet, rootConfig.Normal)
	case "connect":
		return HandleConnect(config, adtClient, rootConfig.Quiet, rootConfig.Normal)
	case "help":
		return handleHelp(config)
	case "create":
		return HandleCreate(config, adtClient, rootConfig.Quiet, rootConfig.Normal)
	case "update":
		return HandleUpdate(config, adtClient, rootConfig.Quiet, rootConfig.Normal)
	case "":
		return fmt.Errorf("action required. Try '%s --help' for usage information", PROGRAM_NAME)
	default:
		return fmt.Errorf("unknown action: %s. Try '%s --help' for available actions", config.Action, PROGRAM_NAME)
	}
}

// handleHelp shows help information
func handleHelp(config *CommandConfig) error {
	if config.ObjectType != "" {
		// Show specific command help
		return showCommandHelp(config.ObjectType)
	}

	rootCmd.Help()
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

func init() {
	// Initialize configuration with defaults
	rootConfig.Mode = "cli"
	rootConfig.Port = "8080"
	rootConfig.ADTHost = set_host()
	rootConfig.ADTClient = os.Getenv("SAP_CLIENT")
	rootConfig.ADTUsername = os.Getenv("SAP_USERNAME")
	rootConfig.ADTPassword = os.Getenv("SAP_PASSWORD")
	rootConfig.Quiet = true // DEFAULT TO QUIET MODE
	rootConfig.LogFile = os.Getenv("ABAPER_LOG_FILE")

	// Add persistent flags
	rootCmd.PersistentFlags().BoolVarP(&rootConfig.Quiet, "quiet", "q", true, "Quiet mode (DEFAULT - minimal CLI output)")
	rootCmd.PersistentFlags().BoolVar(&rootConfig.Normal, "normal", false, "Normal mode (show standard output)")
	rootCmd.PersistentFlags().BoolVarP(&rootConfig.Verbose, "verbose", "v", false, "Verbose mode (detailed output + debug info)")
	rootCmd.PersistentFlags().StringVar(&rootConfig.LogFile, "log-file", rootConfig.LogFile, "Log to specified file (auto-creates directory)")
	rootCmd.PersistentFlags().StringVar(&rootConfig.ADTHost, "adt-host", rootConfig.ADTHost, "SAP system host (or set SAP_HOST)")
	rootCmd.PersistentFlags().StringVar(&rootConfig.ADTClient, "adt-client", rootConfig.ADTClient, "SAP client (or set SAP_CLIENT)")
	rootCmd.PersistentFlags().StringVar(&rootConfig.ADTUsername, "adt-username", rootConfig.ADTUsername, "SAP username (or set SAP_USERNAME)")
	rootCmd.PersistentFlags().StringVar(&rootConfig.ADTPassword, "adt-password", rootConfig.ADTPassword, "SAP password (or set SAP_PASSWORD)")
	rootCmd.PersistentFlags().StringVar(&rootConfig.ConfigFile, "config", "", "Configuration file path")

	// Server command flags
	serverCmd.Flags().StringVarP(&rootConfig.Port, "port", "p", "8080", "Port for server mode")

	// Add subcommands
	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(connectCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(updateCmd)

	// Customize version template
	rootCmd.SetVersionTemplate(`{{.Use}} {{.Version}}
Built: ` + BuildTime + `
Commit: ` + GitCommit + `
Mode: ` + BuildMode + `
Features: CLI and REST Services (No AI)
POSIX Compliant: Yes
`)
}
