package cmd

import (
	"fmt"
	"os"

	"github.com/jiikko/fdup/internal/version"
	"github.com/spf13/cobra"
)

var (
	quiet   bool
	verbose bool
)

var rootCmd = &cobra.Command{
	Use:     "fdup",
	Short:   "File duplicate finder based on filename patterns",
	Version: version.Version,
	Long: `fdup extracts IDs/codes from filenames using regex patterns,
indexes them in SQLite, and detects duplicates across directories.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Print banner unless it's help or version command
		if cmd.Name() != "help" && cmd.Name() != "fdup" && cmd.Name() != "version" {
			if !cmd.Flags().Changed("help") && !cmd.Flags().Changed("version") {
				printBanner()
			}
		}
	},
}

func printBanner() {
	fmt.Fprintf(os.Stderr, "fdup v%s\n", version.Version)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress output")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "", false, "Show detailed output")

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(testCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(dupCmd)
}
