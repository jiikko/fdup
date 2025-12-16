package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jiikko/fdup/internal/config"
	"github.com/jiikko/fdup/internal/db"
	"github.com/jiikko/fdup/internal/scanner"
	"github.com/spf13/cobra"
)

var (
	showProgress bool
	dropDB       bool
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan files and update the index",
	Long:  `Scans the current directory recursively and indexes files matching patterns.`,
	RunE:  runScan,
}

func init() {
	scanCmd.Flags().BoolVarP(&showProgress, "progress", "p", false, "Show progress bar")
	scanCmd.Flags().BoolVarP(&dropDB, "drop", "d", false, "Drop and recreate database")
}

func runScan(cmd *cobra.Command, args []string) error {
	// Find config directory
	configDir, err := config.FindConfigDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(2)
	}

	// Load config
	cfg, err := config.Load(configDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: invalid config.yaml:", err)
		os.Exit(3)
	}

	// Open database
	dbPath := filepath.Join(configDir, config.DBFile)

	// Drop and recreate database if requested
	if dropDB {
		if !quiet {
			fmt.Println("Dropping database...")
		}
		if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove database: %w", err)
		}
	}

	database, err := db.Open(dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: failed to open database:", err)
		os.Exit(4)
	}
	defer func() { _ = database.Close() }()

	// Initialize tables if dropped or new
	if dropDB {
		if err := database.Initialize(); err != nil {
			return fmt.Errorf("failed to initialize database: %w", err)
		}
	}

	if !quiet {
		fmt.Println("Clearing index...")
	}

	// Clear existing index (always full re-index)
	if err := database.Clear(); err != nil {
		return fmt.Errorf("failed to clear index: %w", err)
	}

	// Get root directory (parent of .fdup)
	rootDir := filepath.Dir(configDir)

	// Create scanner
	s, err := scanner.New(cfg.GetPatternRegexes(), cfg.Ignore, rootDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: invalid patterns:", err)
		os.Exit(3)
	}

	if !quiet {
		fmt.Println("Scanning...")
	}

	// Progress callback
	var progressFn scanner.ProgressFunc
	if showProgress && !quiet {
		progressFn = func(current, total int) {
			pct := float64(current) / float64(total) * 100
			bar := makeProgressBar(pct, 40)
			fmt.Fprintf(os.Stderr, "\rScanning... [%s] %.0f%% (%d/%d files)", bar, pct, current, total)
		}
	}

	// Scan files
	records, result, err := s.Scan(progressFn)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	if showProgress && !quiet {
		fmt.Fprintln(os.Stderr) // New line after progress bar
	}

	// Insert records
	for _, rec := range records {
		if err := database.InsertFile(rec); err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "Warning: failed to insert %s: %v\n", rec.Path, err)
			}
		}
	}

	if !quiet {
		fmt.Printf("Found %d files\n", result.TotalFiles)
		fmt.Printf("Added %d new records\n", result.AddedFiles)
	}

	if verbose && len(result.Errors) > 0 {
		fmt.Fprintf(os.Stderr, "\nErrors (%d):\n", len(result.Errors))
		for _, e := range result.Errors {
			fmt.Fprintf(os.Stderr, "  - %v\n", e)
		}
	}

	return nil
}

func makeProgressBar(pct float64, width int) string {
	filled := int(pct / 100 * float64(width))
	bar := make([]byte, width)
	for i := 0; i < width; i++ {
		if i < filled {
			bar[i] = '#'
		} else {
			bar[i] = '-'
		}
	}
	return string(bar)
}
