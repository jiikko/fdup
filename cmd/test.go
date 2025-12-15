package cmd

import (
	"fmt"
	"os"

	"github.com/koji/fdup/internal/config"
	"github.com/koji/fdup/internal/scanner"
	"github.com/spf13/cobra"
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Test patterns against config test cases",
	Long:  `Validates that patterns in config.yaml work as expected using defined test cases.`,
	RunE:  runTest,
}

func runTest(cmd *cobra.Command, args []string) error {
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

	if len(cfg.Test) == 0 {
		fmt.Println("No test cases defined in config.yaml")
		return nil
	}

	// Create scanner to use its extractor
	s, err := scanner.New(cfg.GetPatternRegexes(), nil, ".")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: invalid patterns:", err)
		os.Exit(3)
	}

	if !quiet {
		fmt.Println("Testing patterns...")
	}

	passed := 0
	failed := 0

	for _, tc := range cfg.Test {
		extracted, found := s.ExtractCode(tc.Input)
		var result string
		var ok bool

		if tc.Expected == nil {
			// Expect no match
			ok = !found
			if found {
				result = fmt.Sprintf("expected (no match), got %s", extracted)
			} else {
				result = "(no match)"
			}
		} else {
			// Expect specific match
			ok = found && extracted == *tc.Expected
			if !found {
				result = fmt.Sprintf("expected %s, got (no match)", *tc.Expected)
			} else if extracted != *tc.Expected {
				result = fmt.Sprintf("expected %s, got %s", *tc.Expected, extracted)
			} else {
				result = extracted
			}
		}

		if ok {
			passed++
			if !quiet {
				fmt.Printf("✓ %s -> %s\n", tc.Input, result)
			}
		} else {
			failed++
			fmt.Printf("✗ %s -> %s\n", tc.Input, result)
		}
	}

	total := len(cfg.Test)
	if failed > 0 {
		fmt.Printf("%d of %d tests failed.\n", failed, total)
		os.Exit(1)
	}

	if !quiet {
		fmt.Printf("All %d tests passed.\n", total)
	}

	return nil
}
