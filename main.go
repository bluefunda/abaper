package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/bluefunda/abaper/rest/server"
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
	Version   string = "v0.0.1"
	BuildTime string = "unknown"
	GitCommit string = "unknown"
	BuildMode string = "dev"
)

// Configuration with file logging and quiet default
type Config struct {
	// Mode
	Mode string // "server", "cli"
	Port string

	// Common flags - QUIET IS NOW DEFAULT
	Quiet   bool // Default: true
	Verbose bool
	Help    bool
	Version bool

	// ADT Configuration
	ADTHost     string
	ADTClient   string
	ADTUsername string
	ADTPassword string

	// CLI specific
	Action     string // get, connect, search, list
	ObjectType string // program, class, function, etc.
	ObjectName string
	Args       []string // Additional arguments
	ConfigFile string

	// File logging support
	LogFile string
}

const (
	PROGRAM_NAME = "abaper"
)

var logger *zap.Logger

// Global ADT client cache for connection reuse
var (
	cachedADTClient *ADTClient
	cachedADTConfig string
	cacheTime       time.Time
	cacheTimeout    = 30 * time.Minute
)

func set_host() string {
	sapHost := os.Getenv("SAP_HOST")
	sapPort := os.Getenv("SAP_PORT")
	if sapHost != "" && sapPort != "" {
		return sapHost + ":" + sapPort
	} else {
		return os.Getenv("SAP_HOST") // fallback to original behavior
	}
}

// Enhanced argument parser with proper --key=value support
func parseArgs(args []string) (*Config, error) {
	config := &Config{
		Mode: "cli", // Default to CLI mode
		Port: "8013",
		// ADT defaults from environment
		ADTHost:     set_host(),
		ADTClient:   os.Getenv("SAP_CLIENT"),
		ADTUsername: os.Getenv("SAP_USERNAME"),
		ADTPassword: os.Getenv("SAP_PASSWORD"),
		// DEFAULT TO QUIET MODE for minimal CLI output
		Quiet:   true, // This is the key change - default to quiet
		Verbose: false,
		LogFile: os.Getenv("ABAPER_LOG_FILE"), // Support env var
	}

	i := 1 // Skip program name
	for i < len(args) {
		arg := args[i]

		// Handle --key=value format
		if strings.Contains(arg, "=") {
			parts := strings.SplitN(arg, "=", 2)
			key := parts[0]
			value := parts[1]

			switch key {
			case "--log-file":
				config.LogFile = value
			case "--adt-host":
				config.ADTHost = value
			case "--adt-client":
				config.ADTClient = value
			case "--adt-username":
				config.ADTUsername = value
			case "--adt-password":
				config.ADTPassword = value
			case "--config":
				config.ConfigFile = value
			case "-p", "--port":
				config.Port = value
			default:
				return nil, fmt.Errorf("unknown option: %s", key)
			}
			i++
			continue
		}

		// Handle regular arguments
		switch arg {
		case "-h", "--help":
			config.Help = true
			return config, nil

		case "-v", "--version":
			config.Version = true
			return config, nil

		case "-q", "--quiet":
			config.Quiet = true
			config.Verbose = false

		case "-V", "--verbose":
			config.Verbose = true
			config.Quiet = false

		case "--normal":
			// Allow switching from default quiet to normal mode
			config.Quiet = false
			config.Verbose = false

		case "--server":
			config.Mode = "server"

		case "-p", "--port":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("option '%s' requires an argument", arg)
			}
			i++
			config.Port = args[i]

		case "--log-file":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("option '%s' requires an argument", arg)
			}
			i++
			config.LogFile = args[i]

		case "--adt-host":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("option '%s' requires an argument", arg)
			}
			i++
			config.ADTHost = args[i]

		case "--adt-client":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("option '%s' requires an argument", arg)
			}
			i++
			config.ADTClient = args[i]

		case "--adt-username":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("option '%s' requires an argument", arg)
			}
			i++
			config.ADTUsername = args[i]

		case "--adt-password":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("option '%s' requires an argument", arg)
			}
			i++
			config.ADTPassword = args[i]

		case "--config":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("option '%s' requires an argument", arg)
			}
			i++
			config.ConfigFile = args[i]

		default:
			if strings.HasPrefix(arg, "-") {
				return nil, fmt.Errorf("unknown option: %s", arg)
			}

			// Parse POSIX-style command: <action> <type> [<n>] [args...]
			if config.Action == "" {
				config.Action = arg
			} else if config.ObjectType == "" {
				config.ObjectType = arg
			} else if config.ObjectName == "" {
				config.ObjectName = arg
			} else {
				config.Args = append(config.Args, arg)
			}
		}
		i++
	}

	return config, nil
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

// Enhanced help message
func printHelp() {
	fmt.Printf(`Usage: %s [OPTIONS] ACTION TYPE [NAME] [ARGS...]

ABAP Development Tool - CLI and REST Services (No AI)

ACTIONS:
  get         Retrieve ABAP object source code
  search      Search for ABAP objects
  connect     Test ADT connection
  list        List objects (packages, etc.)
  help        Show help information

OBJECT TYPES:
  program     ABAP program/report
  class       ABAP class
  function    ABAP function module (requires function group)
  include     ABAP include
  interface   ABAP interface
  structure   ABAP structure
  table       ABAP table
  package     ABAP package

OPTIONS:
  -h, --help               Show this help message and exit
  -v, --version            Show version information and exit
  -q, --quiet              Quiet mode (DEFAULT - minimal CLI output)
      --normal             Normal mode (show standard output)
  -V, --verbose            Verbose mode (detailed output + debug info)
      --log-file=FILE      Log to specified file (auto-creates directory)
      --log-file FILE      Log to specified file (space-separated format)
      --server             Run as REST API server
  -p, --port PORT      	   Port for server mode (default: 8013)
      --adt-host=HOST      SAP system host (or set SAP_HOST)
      --adt-client=CLIENT  SAP client (or set SAP_CLIENT)
      --adt-username=USER  SAP username (or set SAP_USERNAME)
      --adt-password=PASS  SAP password (or set SAP_PASSWORD)
      --config=FILE    	   Configuration file path

EXAMPLES:
  # Default quiet mode
  %s get program ZTEST

  # Quiet with file logging (both formats work)
  %s --log-file=./logs/abaper.log get program ZTEST
  %s --log-file ./logs/abaper.log get program ZTEST

  # Normal mode with standard output
  %s --normal get class ZCL_TEST

  # Verbose debugging
  %s --verbose --log-file=./debug.log connect

ENVIRONMENT VARIABLES:
  SAP_HOST            SAP system host
  SAP_CLIENT          SAP client number
  SAP_USERNAME        SAP username
  SAP_PASSWORD        SAP password
  ABAPER_LOG_FILE     Default log file path

EXIT STATUS:
  0    Success
  1    General error
  2    Invalid usage
  130  Interrupted by user (Ctrl+C)
  143  Terminated by signal

Report bugs to: https://github.com/bluefunda/abaper/issues
Organization: BlueFunda, Inc.
`, PROGRAM_NAME, PROGRAM_NAME, PROGRAM_NAME, PROGRAM_NAME, PROGRAM_NAME, PROGRAM_NAME)
}

// Version info
func printVersion() {
	fmt.Printf("%s %s\n", PROGRAM_NAME, Version)
	fmt.Printf("Built: %s\n", BuildTime)
	fmt.Printf("Commit: %s\n", GitCommit)
	fmt.Printf("Mode: %s\n", BuildMode)
	fmt.Printf("Features: CLI and REST Services (No AI)\n")
	fmt.Printf("POSIX Compliant: Yes\n")
}

// getCachedADTClient returns cached client if valid, creates new one otherwise
func getCachedADTClient(config *Config) (*ADTClient, error) {
	// Create cache key from config
	configKey := fmt.Sprintf("%s|%s|%s|%s",
		config.ADTHost, config.ADTClient, config.ADTUsername, config.ADTPassword)

	// Check if we have a valid cached client
	if cachedADTClient != nil &&
		cachedADTConfig == configKey &&
		time.Since(cacheTime) < cacheTimeout &&
		cachedADTClient.IsAuthenticated() {

		// Optional: Test connection with lightweight ping (can be disabled for performance)
		if err := cachedADTClient.ping(); err != nil {
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

	client, err := createADTClient(config)
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

// Error handling helper
func exitWithError(err error, exitCode int) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", PROGRAM_NAME, err)
	if logger != nil {
		logger.Error("Application error", zap.Error(err), zap.Int("exit_code", exitCode))
		logger.Sync()
	}
	os.Exit(exitCode)
}

// runServerMode starts the REST server with CLI and ADT integration
func runServerMode(config *Config) {
	logger.Info("Starting in server mode", zap.String("port", config.Port))

	serverConfig := &server.Config{
		ADTHost:     config.ADTHost,
		ADTClient:   config.ADTClient,
		ADTUsername: config.ADTUsername,
		ADTPassword: config.ADTPassword,
		Verbose:     config.Verbose,
		Quiet:       config.Quiet,
	}
	restServer := server.NewRestServer(serverConfig, logger)
	restServer.Start(config.Port)
}

// runCLIMode starts the CLI interface
func runCLIMode(config *Config) {
	logger.Info("Starting in CLI mode",
		zap.String("action", config.Action),
		zap.String("object_type", config.ObjectType),
		zap.String("object_name", config.ObjectName))

	// Create ADT client for commands that need it
	var adtClient *ADTClient
	var err error

	needsADT := map[string]bool{
		"get": true, "search": true, "list": true, "connect": true,
	}

	if needsADT[config.Action] {
		// Use cached ADT client for better performance
		adtClient, err = getCachedADTClient(config)
		if err != nil {
			if config.Action == "connect" {
				// For connect command, show the error but continue to demonstrate the problem
				fmt.Printf("❌ ADT connection failed: %v\n", err)
				os.Exit(ExitGeneralError)
			} else {
				exitWithError(fmt.Errorf("failed to create ADT client: %w", err), ExitGeneralError)
			}
		}
	}

	// Execute the POSIX command
	if err := executeCommand(config, adtClient); err != nil {
		exitWithError(err, ExitGeneralError)
	}
}

// Execute POSIX-style command
func executeCommand(config *Config, adtClient *ADTClient) error {
	switch config.Action {
	case "get":
		return handleGet(config, adtClient)
	case "search":
		return handleSearch(config, adtClient)
	case "list":
		return handleList(config, adtClient)
	case "connect":
		return handleConnect(config, adtClient)
	case "help":
		return handleHelp(config)
	case "":
		if config.Mode == "cli" {
			return fmt.Errorf("action required. Try '%s help' for usage information", PROGRAM_NAME)
		}
		return nil
	default:
		return fmt.Errorf("unknown action: %s. Try '%s help' for available actions", config.Action, PROGRAM_NAME)
	}
}

// Main function
func main() {
	// Parse command line arguments
	config, err := parseArgs(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", PROGRAM_NAME, err)
		fmt.Fprintf(os.Stderr, "Try '%s --help' for more information.\n", PROGRAM_NAME)
		os.Exit(ExitMisuse)
	}

	// Handle help and version first
	if config.Help {
		printHelp()
		os.Exit(ExitSuccess)
	}

	if config.Version {
		printVersion()
		os.Exit(ExitSuccess)
	}

	// Initialize logger
	initLogger(config.Verbose, config.Quiet, config.LogFile)
	defer logger.Sync()

	// Setup ADT cache cleanup on exit
	defer cleanupADTCache()

	// Setup signal handling
	setupSignalHandling()

	// Log startup
	logger.Info("Application starting",
		zap.String("version", Version),
		zap.String("build_mode", BuildMode),
		zap.String("build_time", BuildTime),
		zap.String("git_commit", GitCommit),
		zap.String("mode", config.Mode))

	// Route to appropriate mode
	switch config.Mode {
	case "server":
		runServerMode(config)
	case "cli":
		runCLIMode(config)
	default:
		exitWithError(fmt.Errorf("unknown mode: %s", config.Mode), ExitMisuse)
	}
}
