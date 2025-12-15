package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// DB wraps the SQLite database connection.
type DB struct {
	conn *sql.DB
}

// FileRecord represents a file record in the database.
type FileRecord struct {
	Path      string
	Code      string
	Size      int64
	Mtime     time.Time
	CreatedAt time.Time
}

// DuplicateGroup represents a group of duplicate files.
type DuplicateGroup struct {
	Code  string
	Files []FileRecord
}

// Open opens or creates the database at the given path.
func Open(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	return &DB{conn: conn}, nil
}

// Close closes the database connection.
func (d *DB) Close() error {
	return d.conn.Close()
}

// Initialize creates the database tables.
func (d *DB) Initialize() error {
	schema := `
		CREATE TABLE IF NOT EXISTS codes (
			code TEXT PRIMARY KEY,
			created_at DATETIME
		);

		CREATE TABLE IF NOT EXISTS files (
			path TEXT PRIMARY KEY,
			code TEXT REFERENCES codes(code),
			size INTEGER,
			mtime DATETIME,
			created_at DATETIME
		);

		CREATE INDEX IF NOT EXISTS idx_code ON files(code);
		CREATE INDEX IF NOT EXISTS idx_size ON files(size);
	`
	_, err := d.conn.Exec(schema)
	return err
}

// Clear removes all records from the database.
func (d *DB) Clear() error {
	_, err := d.conn.Exec("DELETE FROM files; DELETE FROM codes;")
	return err
}

// InsertFile inserts or updates a file record.
func (d *DB) InsertFile(record FileRecord) error {
	now := time.Now()

	// Ensure code exists
	_, err := d.conn.Exec(`
		INSERT OR IGNORE INTO codes (code, created_at)
		VALUES (?, ?)
	`, record.Code, now)
	if err != nil {
		return err
	}

	// Insert file
	_, err = d.conn.Exec(`
		INSERT OR REPLACE INTO files (path, code, size, mtime, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, record.Path, record.Code, record.Size, record.Mtime, now)
	return err
}

// SearchByCode searches for files by code prefix or exact match.
func (d *DB) SearchByCode(code string, exact bool) ([]DuplicateGroup, error) {
	var query string
	var args []interface{}

	if exact {
		query = `
			SELECT f.path, f.code, f.size, f.mtime, f.created_at
			FROM files f
			WHERE f.code = ?
			ORDER BY f.code, f.path
		`
		args = []interface{}{code}
	} else {
		query = `
			SELECT f.path, f.code, f.size, f.mtime, f.created_at
			FROM files f
			WHERE f.code LIKE ?
			ORDER BY f.code, f.path
		`
		args = []interface{}{code + "%"}
	}

	rows, err := d.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return groupResults(rows)
}

// FindDuplicates finds all duplicate file groups.
// Duplicates are files with the same code but in different directories.
func (d *DB) FindDuplicates() ([]DuplicateGroup, error) {
	// First, get all files grouped by code
	query := `
		SELECT f.path, f.code, f.size, f.mtime, f.created_at
		FROM files f
		WHERE f.code IN (
			SELECT code FROM files GROUP BY code HAVING COUNT(*) > 1
		)
		ORDER BY f.code, f.path
	`

	rows, err := d.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	groups, err := groupResults(rows)
	if err != nil {
		return nil, err
	}

	// Filter to keep only groups with files in different directories
	var result []DuplicateGroup
	for _, group := range groups {
		dirs := make(map[string]bool)
		for _, f := range group.Files {
			dirs[filepath.Dir(f.Path)] = true
		}
		// Only include if files are in different directories
		if len(dirs) > 1 {
			result = append(result, group)
		}
	}

	return result, nil
}

// GetFileCount returns the total number of indexed files.
func (d *DB) GetFileCount() (int, error) {
	var count int
	err := d.conn.QueryRow("SELECT COUNT(*) FROM files").Scan(&count)
	return count, err
}

// DeleteFile removes a file from the database.
func (d *DB) DeleteFile(path string) error {
	_, err := d.conn.Exec("DELETE FROM files WHERE path = ?", path)
	return err
}

// UpdateFilePath updates a file's path (for move operations).
func (d *DB) UpdateFilePath(oldPath, newPath string) error {
	_, err := d.conn.Exec("UPDATE files SET path = ? WHERE path = ?", newPath, oldPath)
	return err
}

func groupResults(rows *sql.Rows) ([]DuplicateGroup, error) {
	groups := make(map[string][]FileRecord)
	order := []string{}

	for rows.Next() {
		var rec FileRecord
		if err := rows.Scan(&rec.Path, &rec.Code, &rec.Size, &rec.Mtime, &rec.CreatedAt); err != nil {
			return nil, err
		}
		if _, exists := groups[rec.Code]; !exists {
			order = append(order, rec.Code)
		}
		groups[rec.Code] = append(groups[rec.Code], rec)
	}

	result := make([]DuplicateGroup, 0, len(groups))
	for _, code := range order {
		result = append(result, DuplicateGroup{
			Code:  code,
			Files: groups[code],
		})
	}
	return result, rows.Err()
}

// GetDirectory extracts the directory from a file path.
func GetDirectory(path string) string {
	return filepath.Dir(path)
}

// FileExists checks if a file exists on disk.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
