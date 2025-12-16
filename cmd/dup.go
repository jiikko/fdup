package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jiikko/fdup/internal/code"
	"github.com/jiikko/fdup/internal/config"
	"github.com/jiikko/fdup/internal/db"
	"github.com/jiikko/fdup/internal/tui"
	"github.com/jiikko/fdup/internal/web"
	"github.com/spf13/cobra"
)

var (
	interactive bool
	dryRun      bool
	useTrash    bool
	webMode     bool
)

var dupCmd = &cobra.Command{
	Use:   "dup",
	Short: "Find duplicate files",
	Long:  `Lists files with the same code in different directories.`,
	RunE:  runDup,
}

func init() {
	dupCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Interactive TUI mode")
	dupCmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Show what would be done without making changes")
	dupCmd.Flags().BoolVarP(&useTrash, "trash", "t", false, "Move to trash instead of deleting")
	dupCmd.Flags().BoolVarP(&webMode, "web", "w", false, "Web UI mode")
}

func runDup(cmd *cobra.Command, args []string) error {
	// Find config directory
	configDir, err := config.FindConfigDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(2)
	}

	// Load config (as per spec flow)
	_, err = config.Load(configDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: invalid config.yaml:", err)
		os.Exit(3)
	}

	// Open database
	dbPath := filepath.Join(configDir, config.DBFile)
	database, err := db.Open(dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: failed to open database:", err)
		os.Exit(4)
	}
	defer func() { _ = database.Close() }()

	// Check if index is empty
	count, err := database.GetFileCount()
	if err != nil {
		return fmt.Errorf("failed to check index: %w", err)
	}
	if count == 0 {
		if !quiet {
			fmt.Println("No files indexed. Run 'fdup scan' first")
		}
		return nil
	}

	// Find duplicates
	groups, err := database.FindDuplicates()
	if err != nil {
		return fmt.Errorf("failed to find duplicates: %w", err)
	}

	if len(groups) == 0 {
		if !quiet {
			fmt.Println("No duplicates found")
		}
		return nil
	}

	if interactive {
		return tui.Run(groups, database, dryRun, useTrash)
	}

	if webMode {
		return web.Run(groups, database)
	}

	// Basic text output
	for _, group := range groups {
		fileWord := "files"
		if len(group.Files) == 1 {
			fileWord = "file"
		}
		fmt.Printf("%s: %d %s\n", code.Format(group.Code), len(group.Files), fileWord)
		for _, f := range group.Files {
			fmt.Printf("  %s (%s)\n", f.Path, formatSize(f.Size))
		}
		fmt.Println()
	}

	return nil
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
