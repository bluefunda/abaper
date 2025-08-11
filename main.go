package main

import (
	"os"

	"go.uber.org/zap"
)

// Main function
func main() {
	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		if logger != nil {
			logger.Error("Command execution failed", zap.Error(err))
			logger.Sync()
		}
		os.Exit(ExitGeneralError)
	}
}
