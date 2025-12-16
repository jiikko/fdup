package scanner

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/jiikko/fdup/internal/code"
	"github.com/jiikko/fdup/internal/db"
)

// Scanner scans directories for files matching patterns.
type Scanner struct {
	extractor    *code.Extractor
	ignorePatterns []string
	rootDir      string
}

// ScanResult contains the results of a scan operation.
type ScanResult struct {
	TotalFiles int
	AddedFiles int
	Errors     []error
}

// ProgressFunc is called during scanning to report progress.
type ProgressFunc func(current, total int)

// New creates a new scanner with the given patterns and ignore rules.
func New(patterns []string, ignore []string, rootDir string) (*Scanner, error) {
	extractor, err := code.NewExtractor(patterns)
	if err != nil {
		return nil, err
	}
	return &Scanner{
		extractor:    extractor,
		ignorePatterns: ignore,
		rootDir:      rootDir,
	}, nil
}

// Scan scans the directory and returns file records.
func (s *Scanner) Scan(progress ProgressFunc) ([]db.FileRecord, *ScanResult, error) {
	var files []string
	var errors []error

	// First pass: collect all files
	err := filepath.WalkDir(s.rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			errors = append(errors, err)
			return nil
		}

		relPath, _ := filepath.Rel(s.rootDir, path)
		if s.shouldIgnore(relPath, d.IsDir()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip symlinks
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}

		if !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	// Second pass: extract codes and create records
	var records []db.FileRecord
	total := len(files)

	for i, path := range files {
		if progress != nil {
			progress(i+1, total)
		}

		filename := filepath.Base(path)
		normalized, found := s.extractor.Extract(filename)
		if !found {
			continue
		}

		info, err := os.Stat(path)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		absPath, err := filepath.Abs(path)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		records = append(records, db.FileRecord{
			Path:  absPath,
			Code:  normalized,
			Size:  info.Size(),
			Mtime: info.ModTime(),
		})
	}

	result := &ScanResult{
		TotalFiles: total,
		AddedFiles: len(records),
		Errors:     errors,
	}

	return records, result, nil
}

// shouldIgnore checks if a path should be ignored based on patterns.
func (s *Scanner) shouldIgnore(relPath string, isDir bool) bool {
	// Split path into components for matching
	parts := strings.Split(relPath, string(filepath.Separator))

	for _, pattern := range s.ignorePatterns {
		// Directory pattern (ends with /)
		if strings.HasSuffix(pattern, "/") {
			dirPattern := strings.TrimSuffix(pattern, "/")
			// Check if any path component matches the directory pattern
			for _, part := range parts {
				if matchPattern(part, dirPattern) {
					return true
				}
			}
			continue
		}

		// File pattern - match against full path and each component
		if matchPattern(relPath, pattern) {
			return true
		}
		// Match against filename
		if matchPattern(filepath.Base(relPath), pattern) {
			return true
		}
		// Match against any path component (for patterns like .git)
		for _, part := range parts {
			if matchPattern(part, pattern) {
				return true
			}
		}
	}
	return false
}

// matchPattern does simple glob matching (* wildcard).
func matchPattern(name, pattern string) bool {
	// Handle simple * wildcard
	if strings.Contains(pattern, "*") {
		matched, _ := filepath.Match(pattern, name)
		return matched
	}
	return name == pattern
}

// ExtractCode extracts a code from a filename using the scanner's extractor.
func (s *Scanner) ExtractCode(filename string) (string, bool) {
	return s.extractor.Extract(filename)
}
