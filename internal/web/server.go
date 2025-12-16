package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/jiikko/fdup/internal/db"
)

// Server holds the web server state.
type Server struct {
	groups   []db.DuplicateGroup
	database *db.DB
	server   *http.Server
	port     int
}

// Run starts the web server and opens the browser.
func Run(groups []db.DuplicateGroup, database *db.DB) error {
	s := &Server{
		groups:   groups,
		database: database,
	}

	// Find available port starting from 8080
	port, listener, err := findAvailablePort(8080)
	if err != nil {
		return fmt.Errorf("failed to find available port: %w", err)
	}
	s.port = port

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/open", s.handleOpen)
	mux.HandleFunc("/api/reveal", s.handleReveal)
	mux.HandleFunc("/api/delete", s.handleDelete)
	mux.HandleFunc("/api/shutdown", s.handleShutdown)

	s.server = &http.Server{
		Handler: mux,
	}

	// Handle graceful shutdown
	done := make(chan bool)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		fmt.Println("\nShutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.server.Shutdown(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Server shutdown error: %v\n", err)
		}
		close(done)
	}()

	// Open browser
	url := fmt.Sprintf("http://localhost:%d", port)
	fmt.Printf("Starting web server at %s\n", url)
	fmt.Println("Press Ctrl+C to stop")

	go func() {
		time.Sleep(500 * time.Millisecond)
		openBrowser(url)
	}()

	// Start server
	if err := s.server.Serve(listener); err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	<-done
	return nil
}

func findAvailablePort(startPort int) (int, net.Listener, error) {
	for port := startPort; port < startPort+100; port++ {
		addr := fmt.Sprintf(":%d", port)
		listener, err := net.Listen("tcp", addr)
		if err == nil {
			return port, listener, nil
		}
	}
	return 0, nil, fmt.Errorf("no available port found")
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return
	}
	_ = cmd.Start()
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(s.renderHTML()))
}

func (s *Server) handleOpen(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if err := openFile(req.Path); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonSuccess(w, "File opened")
}

func (s *Server) handleReveal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if err := revealInFinder(req.Path); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonSuccess(w, "Revealed in Finder")
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if err := moveToTrash(req.Path); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Remove from database
	if s.database != nil {
		_ = s.database.DeleteFile(req.Path)
	}

	jsonSuccess(w, "Moved to trash")
}

func (s *Server) handleShutdown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	jsonSuccess(w, "Server shutting down")

	go func() {
		time.Sleep(100 * time.Millisecond)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.server.Shutdown(ctx)
	}()
}

func openFile(path string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", path).Run()
	case "linux":
		return exec.Command("xdg-open", path).Run()
	default:
		return fmt.Errorf("open file not supported on %s", runtime.GOOS)
	}
}

func revealInFinder(path string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", "-R", path).Run()
	case "linux":
		dir := filepath.Dir(path)
		return exec.Command("xdg-open", dir).Run()
	default:
		return fmt.Errorf("reveal in finder not supported on %s", runtime.GOOS)
	}
}

func moveToTrash(path string) error {
	var trashDir string
	switch runtime.GOOS {
	case "darwin":
		home, _ := os.UserHomeDir()
		trashDir = filepath.Join(home, ".Trash")
	case "linux":
		home, _ := os.UserHomeDir()
		trashDir = filepath.Join(home, ".local", "share", "Trash", "files")
	default:
		return os.Remove(path)
	}

	if err := os.MkdirAll(trashDir, 0755); err != nil {
		return err
	}

	destPath := filepath.Join(trashDir, filepath.Base(path))
	return os.Rename(path, destPath)
}

func jsonSuccess(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "message": message})
}

func jsonError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": message})
}
