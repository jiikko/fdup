package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/koji/fdup/internal/config"
	"github.com/koji/fdup/internal/db"
	"github.com/spf13/cobra"
)

var forceInit bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize fdup in the current directory",
	Long:  `Creates the .fdup/ directory with config.yaml and fdup.db`,
	RunE:  runInit,
}

func init() {
	initCmd.Flags().BoolVarP(&forceInit, "force", "f", false, "Force reinitialize (deletes existing data)")
}

func runInit(cmd *cobra.Command, args []string) error {
	configDir := filepath.Join(".", config.DirName)

	// Check if already initialized
	if _, err := os.Stat(configDir); err == nil {
		if !forceInit {
			return fmt.Errorf("already initialized. Use --force to reinitialize")
		}
		// Remove existing directory
		if err := os.RemoveAll(configDir); err != nil {
			return fmt.Errorf("failed to remove existing .fdup: %w", err)
		}
	}

	// Create .fdup directory
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create .fdup/: %w", err)
	}
	if !quiet {
		fmt.Println("Created .fdup/")
	}

	// Create config.yaml with defaults
	cfg := config.DefaultConfig()
	if err := config.Save(configDir, cfg); err != nil {
		return fmt.Errorf("failed to create config.yaml: %w", err)
	}
	if !quiet {
		fmt.Println("Created .fdup/config.yaml")
	}

	// Create and initialize database
	dbPath := filepath.Join(configDir, config.DBFile)
	database, err := db.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}
	defer func() { _ = database.Close() }()

	if err := database.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	if !quiet {
		fmt.Println("Created .fdup/fdup.db")
	}

	if !quiet {
		fmt.Println("Initialized successfully.")
	}

	return nil
}
