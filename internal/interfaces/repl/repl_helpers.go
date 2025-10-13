package repl

import (
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// navigateHistory navigates command history via History.Navigator.
func (m Model) navigateHistory(direction string) (tea.Model, tea.Cmd) {
	var cmd string
	var ok bool

	switch direction {
	case "up":
		cmd, ok = m.historyNavigator.Backward()
	case "down":
		cmd, ok = m.historyNavigator.Forward()
	}

	// If navigation successful, set value
	if ok || direction == "down" {
		m.textarea.SetValue(cmd)
		if cmd != "" {
			m.textarea.CursorEnd()
		}
		// Sync input state
		m.inputText = cmd
		m.cursorPos = len([]rune(m.inputText))
	}

	return m, nil
}

// addOutputRaw adds line to output WITHOUT applying styles (for ANSI codes).
func (m *Model) addOutputRaw(line string) {
	m.output = append(m.output, line)

	// Limit output size
	if len(m.output) > m.maxOutputLines {
		m.output = m.output[len(m.output)-m.maxOutputLines:]
	}
}

// updateViewportContent updates viewport content from output.
func (m *Model) updateViewportContent() {
	content := strings.Join(m.output, "\n")
	m.viewport.SetContent(content)
}

// updateGitInfo updates Git repository information.
func (m *Model) updateGitInfo() {
	m.gitBranch = ""
	m.gitDirty = false

	workDir := m.currentSession.WorkingDirectory()

	// Check for .git presence
	gitDir := filepath.Join(workDir, ".git")
	if _, err := filepath.Glob(gitDir); err != nil {
		return
	}

	// Get current branch
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = workDir
	if output, err := cmd.Output(); err == nil {
		m.gitBranch = strings.TrimSpace(string(output))
	}

	// Check for changes
	cmd = exec.Command("git", "status", "--porcelain")
	cmd.Dir = workDir
	if output, err := cmd.Output(); err == nil {
		m.gitDirty = strings.TrimSpace(string(output)) != ""
	}
}
