package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jiikko/fdup/internal/code"
	"github.com/jiikko/fdup/internal/config"
	"github.com/jiikko/fdup/internal/db"
	"github.com/spf13/cobra"
)

var (
	exactMatch bool
	jsonOutput bool
)

var searchCmd = &cobra.Command{
	Use:   "search <CODE>",
	Short: "Search for files by code",
	Long:  `Searches the index for files matching the given code.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runSearch,
}

func init() {
	searchCmd.Flags().BoolVarP(&exactMatch, "exact", "e", false, "Exact match only")
	searchCmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output as JSON")
}

func runSearch(cmd *cobra.Command, args []string) error {
	query := args[0]

	// Find config directory
	configDir, err := config.FindConfigDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(2)
	}

	// Open database
	dbPath := filepath.Join(configDir, config.DBFile)
	database, err := db.Open(dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: failed to open database:", err)
		os.Exit(4)
	}
	defer func() { _ = database.Close() }()

	// Normalize query
	normalizedQuery := code.Normalize(query)

	// Search
	groups, err := database.SearchByCode(normalizedQuery, exactMatch)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(groups) == 0 {
		if !quiet {
			fmt.Printf("No files found for: %s\n", query)
		}
		return nil
	}

	if jsonOutput {
		return outputJSON(groups)
	}

	return outputText(groups)
}

func outputJSON(groups []db.DuplicateGroup) error {
	for _, group := range groups {
		paths := make([]string, len(group.Files))
		for i, f := range group.Files {
			paths[i] = f.Path
		}
		data := map[string]interface{}{
			"code":  code.Format(group.Code),
			"files": paths,
		}
		b, err := json.Marshal(data)
		if err != nil {
			return err
		}
		fmt.Println(string(b))
	}
	return nil
}

func outputText(groups []db.DuplicateGroup) error {
	for _, group := range groups {
		fileWord := "files"
		if len(group.Files) == 1 {
			fileWord = "file"
		}
		fmt.Printf("%s: %d %s\n", code.Format(group.Code), len(group.Files), fileWord)
		for _, f := range group.Files {
			fmt.Printf("  %s\n", f.Path)
		}
	}
	return nil
}
