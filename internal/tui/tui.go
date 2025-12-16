package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jiikko/fdup/internal/code"
	"github.com/jiikko/fdup/internal/db"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212"))

	fileStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("246"))

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82"))
)

type state int

const (
	stateSelectFiles state = iota
	stateSelectAction
	stateCustomPath
	stateConfirm
)

// Model is the Bubble Tea model for interactive mode.
type Model struct {
	groups       []db.DuplicateGroup
	currentGroup int
	selected     map[int]bool
	state        state
	textInput    textinput.Model
	dryRun       bool
	useTrash     bool
	database     *db.DB
	message      string
	done         bool
	err          error
}

// NewModel creates a new TUI model.
func NewModel(groups []db.DuplicateGroup, database *db.DB, dryRun, useTrash bool) Model {
	ti := textinput.New()
	ti.Placeholder = "Enter directory path..."
	ti.Width = 50

	return Model{
		groups:       groups,
		currentGroup: 0,
		selected:     make(map[int]bool),
		state:        stateSelectFiles,
		textInput:    ti,
		dryRun:       dryRun,
		useTrash:     useTrash,
		database:     database,
	}
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.done {
		return m, tea.Quit
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case stateSelectFiles:
			return m.handleSelectFiles(msg)
		case stateSelectAction:
			return m.handleSelectAction(msg)
		case stateCustomPath:
			return m.handleCustomPath(msg)
		}
	}

	return m, nil
}

func (m Model) handleSelectFiles(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "q", "ctrl+c":
		m.done = true
		return m, tea.Quit
	case "s":
		// Skip this group
		m.nextGroup()
		return m, nil
	case "enter":
		if len(m.selected) > 0 {
			m.state = stateSelectAction
		}
		return m, nil
	default:
		// Handle file selection: 1-9 for files 1-9, 0 for file 10, a-z for files 11-36
		idx := -1
		if len(key) == 1 {
			ch := key[0]
			if ch >= '1' && ch <= '9' {
				idx = int(ch - '1') // 1->0, 9->8
			} else if ch == '0' {
				idx = 9 // 0->9 (file 10)
			} else if ch >= 'a' && ch <= 'z' {
				idx = int(ch-'a') + 10 // a->10, z->35
			}
		}
		if idx >= 0 {
			group := m.groups[m.currentGroup]
			if idx < len(group.Files) {
				m.selected[idx] = !m.selected[idx]
			}
		}
		return m, nil
	}
}

func (m Model) handleSelectAction(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	group := m.groups[m.currentGroup]
	key := msg.String()

	switch key {
	case "q", "ctrl+c":
		m.done = true
		return m, tea.Quit
	case "escape":
		m.state = stateSelectFiles
		return m, nil
	case "s":
		m.selected = make(map[int]bool)
		m.state = stateSelectFiles
		m.nextGroup()
		return m, nil
	case "d":
		// Delete selected files
		m.performDelete()
		return m, nil
	case "c":
		// Custom path
		m.textInput.Focus()
		m.state = stateCustomPath
		return m, textinput.Blink
	default:
		// Handle move to file's directory: 1-9, 0, a-z
		idx := -1
		if len(key) == 1 {
			ch := key[0]
			if ch >= '1' && ch <= '9' {
				idx = int(ch - '1')
			} else if ch == '0' {
				idx = 9
			} else if ch >= 'a' && ch <= 'z' {
				idx = int(ch-'a') + 10
			}
		}
		if idx >= 0 && idx < len(group.Files) && !m.selected[idx] {
			destDir := filepath.Dir(group.Files[idx].Path)
			m.performMove(destDir)
		}
		return m, nil
	}
}

func (m Model) handleCustomPath(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		destDir := m.textInput.Value()
		if destDir != "" {
			m.performMove(destDir)
		}
		m.textInput.Reset()
		m.state = stateSelectFiles
		return m, nil
	case "escape":
		m.textInput.Reset()
		m.state = stateSelectAction
		return m, nil
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m *Model) performDelete() {
	group := m.groups[m.currentGroup]
	count := len(m.selected)
	var dryRunMsgs []string

	for idx := range m.selected {
		file := group.Files[idx]
		if m.dryRun {
			if m.useTrash {
				dryRunMsgs = append(dryRunMsgs, fmt.Sprintf("[DRY-RUN] Would trash: %s", file.Path))
			} else {
				dryRunMsgs = append(dryRunMsgs, fmt.Sprintf("[DRY-RUN] Would delete: %s", file.Path))
			}
		} else {
			var err error
			if m.useTrash {
				err = moveToTrash(file.Path)
			} else {
				err = os.Remove(file.Path)
			}
			if err != nil {
				m.err = err
				return
			}
			if m.database != nil {
				_ = m.database.DeleteFile(file.Path)
			}
		}
	}

	if m.dryRun {
		m.message = strings.Join(dryRunMsgs, "\n")
	} else {
		word := "file"
		if count > 1 {
			word = "files"
		}
		m.message = successStyle.Render(fmt.Sprintf("Deleted %d %s", count, word))
	}
	m.selected = make(map[int]bool)
	m.nextGroup()
}

func (m *Model) performMove(destDir string) {
	group := m.groups[m.currentGroup]
	count := len(m.selected)
	var dryRunMsgs []string

	for idx := range m.selected {
		file := group.Files[idx]
		destPath := filepath.Join(destDir, filepath.Base(file.Path))
		if m.dryRun {
			dryRunMsgs = append(dryRunMsgs, fmt.Sprintf("[DRY-RUN] Would move: %s -> %s", file.Path, destDir))
		} else {
			if err := os.MkdirAll(destDir, 0755); err != nil {
				m.err = err
				return
			}
			if err := os.Rename(file.Path, destPath); err != nil {
				m.err = err
				return
			}
			if m.database != nil {
				_ = m.database.UpdateFilePath(file.Path, destPath)
			}
		}
	}

	if m.dryRun {
		m.message = strings.Join(dryRunMsgs, "\n")
	} else {
		word := "file"
		if count > 1 {
			word = "files"
		}
		m.message = successStyle.Render(fmt.Sprintf("Moved %d %s to %s", count, word, destDir))
	}
	m.selected = make(map[int]bool)
	m.nextGroup()
}

func (m *Model) nextGroup() {
	m.currentGroup++
	m.selected = make(map[int]bool)
	m.state = stateSelectFiles
	if m.currentGroup >= len(m.groups) {
		m.done = true
	}
}

// View renders the UI.
func (m Model) View() string {
	if m.done {
		if m.err != nil {
			return errorStyle.Render(fmt.Sprintf("Error: %v\n", m.err))
		}
		return "Done.\n"
	}

	if m.currentGroup >= len(m.groups) {
		return "No more duplicates.\n"
	}

	var b strings.Builder
	group := m.groups[m.currentGroup]

	// Title
	b.WriteString(titleStyle.Render(fmt.Sprintf("%s: %d files", code.Format(group.Code), len(group.Files))))
	b.WriteString("\n")

	// Files
	for i, file := range group.Files {
		prefix := "  "
		style := fileStyle
		if m.selected[i] {
			prefix = "> "
			style = selectedStyle
		}
		size := formatSize(file.Size)
		key := indexToKey(i)
		b.WriteString(style.Render(fmt.Sprintf("%s[%s] %s (%s)", prefix, key, file.Path, size)))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// State-specific UI
	switch m.state {
	case stateSelectFiles:
		b.WriteString(helpStyle.Render("Select files to remove (1-9,0,a-z), [enter] confirm, [s] skip, [q] quit"))
	case stateSelectAction:
		b.WriteString(helpStyle.Render("Action:"))
		b.WriteString("\n")
		for i, file := range group.Files {
			if !m.selected[i] {
				dir := filepath.Dir(file.Path)
				key := indexToKey(i)
				b.WriteString(helpStyle.Render(fmt.Sprintf("  [%s] Move to %s", key, dir)))
				b.WriteString("\n")
			}
		}
		b.WriteString(helpStyle.Render("  [c] Custom directory"))
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("  [d] Delete"))
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("  [s] Skip"))
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("  [q] Quit"))
	case stateCustomPath:
		b.WriteString(m.textInput.View())
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("[enter] confirm, [esc] cancel"))
	}

	if m.message != "" {
		b.WriteString("\n\n")
		b.WriteString(m.message)
	}

	return b.String()
}

// indexToKey converts a 0-based index to a key string for display
// 0-8 -> "1"-"9", 9 -> "0", 10-35 -> "a"-"z"
func indexToKey(idx int) string {
	if idx < 9 {
		return fmt.Sprintf("%d", idx+1)
	} else if idx == 9 {
		return "0"
	} else if idx < 36 {
		return string(rune('a' + idx - 10))
	}
	return fmt.Sprintf("%d", idx+1)
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
		// Fallback: just delete
		return os.Remove(path)
	}

	if err := os.MkdirAll(trashDir, 0755); err != nil {
		return err
	}

	destPath := filepath.Join(trashDir, filepath.Base(path))
	return os.Rename(path, destPath)
}

// Run starts the TUI.
func Run(groups []db.DuplicateGroup, database *db.DB, dryRun, useTrash bool) error {
	if len(groups) == 0 {
		fmt.Println("No duplicates found")
		return nil
	}

	p := tea.NewProgram(NewModel(groups, database, dryRun, useTrash))
	_, err := p.Run()
	return err
}
