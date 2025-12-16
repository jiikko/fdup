package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jiikko/fdup/internal/db"
)

func setupTestDB(t *testing.T) (*db.DB, string) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	if err := database.Initialize(); err != nil {
		t.Fatalf("failed to initialize database: %v", err)
	}
	return database, tmpDir
}

func TestHandleIndex(t *testing.T) {
	database, _ := setupTestDB(t)
	defer database.Close()

	// Add test data
	database.InsertFile(db.FileRecord{
		Path:  "/test/dir1/DSC00001.jpg",
		Code:  "DSC00001",
		Size:  1024,
		Mtime: time.Now(),
	})
	database.InsertFile(db.FileRecord{
		Path:  "/test/dir2/DSC_00001.jpg",
		Code:  "DSC00001",
		Size:  2048,
		Mtime: time.Now(),
	})

	s := &Server{database: database}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	s.handleIndex(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "DSC-00001") {
		t.Error("expected body to contain DSC-00001")
	}
	if !strings.Contains(body, "fdup - Duplicate Files") {
		t.Error("expected body to contain title")
	}
}

func TestHandleIndexPagination(t *testing.T) {
	database, _ := setupTestDB(t)
	defer database.Close()

	s := &Server{database: database}

	// Test page parameter
	req := httptest.NewRequest(http.MethodGet, "/?page=1", nil)
	w := httptest.NewRecorder()

	s.handleIndex(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestHandleIndexNotFound(t *testing.T) {
	database, _ := setupTestDB(t)
	defer database.Close()

	s := &Server{database: database}

	req := httptest.NewRequest(http.MethodGet, "/notfound", nil)
	w := httptest.NewRecorder()

	s.handleIndex(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestHandleOpen(t *testing.T) {
	database, _ := setupTestDB(t)
	defer database.Close()

	s := &Server{database: database}

	// Test method not allowed
	req := httptest.NewRequest(http.MethodGet, "/api/open", nil)
	w := httptest.NewRecorder()

	s.handleOpen(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}

	// Test invalid JSON
	req = httptest.NewRequest(http.MethodPost, "/api/open", strings.NewReader("invalid"))
	w = httptest.NewRecorder()

	s.handleOpen(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestHandleReveal(t *testing.T) {
	database, _ := setupTestDB(t)
	defer database.Close()

	s := &Server{database: database}

	// Test method not allowed
	req := httptest.NewRequest(http.MethodGet, "/api/reveal", nil)
	w := httptest.NewRecorder()

	s.handleReveal(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

func TestHandleDelete(t *testing.T) {
	database, tmpDir := setupTestDB(t)
	defer database.Close()

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Add to database
	database.InsertFile(db.FileRecord{
		Path:  testFile,
		Code:  "TEST001",
		Size:  4,
		Mtime: time.Now(),
	})

	s := &Server{database: database}

	// Test method not allowed
	req := httptest.NewRequest(http.MethodGet, "/api/delete", nil)
	w := httptest.NewRecorder()

	s.handleDelete(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}

	// Test invalid JSON
	req = httptest.NewRequest(http.MethodPost, "/api/delete", strings.NewReader("invalid"))
	w = httptest.NewRecorder()

	s.handleDelete(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	// Test successful delete
	body := `{"path":"` + testFile + `"}`
	req = httptest.NewRequest(http.MethodPost, "/api/delete", strings.NewReader(body))
	w = httptest.NewRecorder()

	s.handleDelete(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("expected status ok, got %s", resp["status"])
	}

	// Verify file was moved (not in original location)
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("expected file to be moved to trash")
	}
}

func TestHandleShutdown(t *testing.T) {
	database, _ := setupTestDB(t)
	defer database.Close()

	s := &Server{database: database}

	// Test method not allowed
	req := httptest.NewRequest(http.MethodGet, "/api/shutdown", nil)
	w := httptest.NewRecorder()

	s.handleShutdown(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

func TestFindAvailablePort(t *testing.T) {
	port, listener, err := findAvailablePort(18080)
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	defer listener.Close()

	if port < 18080 || port >= 18180 {
		t.Errorf("expected port in range 18080-18179, got %d", port)
	}
}

func TestJsonSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	jsonSuccess(w, "test message")

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("expected status ok, got %s", resp["status"])
	}
	if resp["message"] != "test message" {
		t.Errorf("expected message 'test message', got %s", resp["message"])
	}
}

func TestJsonError(t *testing.T) {
	w := httptest.NewRecorder()
	jsonError(w, "error message", http.StatusInternalServerError)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "error" {
		t.Errorf("expected status error, got %s", resp["status"])
	}
}
