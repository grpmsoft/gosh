package repl

import (
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/phoenix-tui/phoenix/tea"
)

// All methods in this file use Bubbletea's MVU (Model-View-Update) pattern,.
// which requires value receivers. The "hugeParam" warnings are false positives.
//
//nolint:gocritic // All Model methods: Bubbletea MVU requires value receivers

const (
	directionUp   = "up"
	directionDown = "down"
)

// navigateHistory navigates command history via History.Navigator.
func (m Model) navigateHistory(direction string) (Model, tea.Cmd) {
	var cmd string
	var ok bool

	switch direction {
	case directionUp:
		cmd, ok = m.historyNavigator.Backward()
	case directionDown:
		cmd, ok = m.historyNavigator.Forward()
	}

	// If navigation successful, set value (respect multilineMode)
	if ok || direction == directionDown {
		if m.multilineMode {
			m.shellTextArea.SetValue(cmd)
		} else {
			m.shellInput.SetValue(cmd)
			m.shellInput.RefreshHighlight()
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
	// Phoenix Viewport uses fluent API (returns new viewport)
	m.viewport = m.viewport.SetContent(content)
}

// isIncomplete checks if command needs more input (unclosed quotes, backslash continuation, etc.).
func (m Model) isIncomplete(cmd string) bool {
	if cmd == "" {
		return false
	}

	// Check for backslash continuation at end
	trimmed := strings.TrimRight(cmd, " \t")
	if strings.HasSuffix(trimmed, "\\") {
		return true
	}

	// Count unescaped quotes
	singleQuotes := 0
	doubleQuotes := 0
	escaped := false

	for _, ch := range cmd {
		if escaped {
			escaped = false
			continue
		}

		switch ch {
		case '\\':
			escaped = true
		case '\'':
			singleQuotes++
		case '"':
			doubleQuotes++
		}
	}

	// If odd number of quotes - unclosed
	if singleQuotes%2 != 0 || doubleQuotes%2 != 0 {
		return true
	}

	// Check for pipe at end (incomplete pipeline)
	if strings.HasSuffix(trimmed, "|") {
		return true
	}

	// Check for unclosed braces/brackets (basic heuristic)
	openBraces := strings.Count(cmd, "{") - strings.Count(cmd, "}")
	openBrackets := strings.Count(cmd, "[") - strings.Count(cmd, "]")
	openParens := strings.Count(cmd, "(") - strings.Count(cmd, ")")

	if openBraces > 0 || openBrackets > 0 || openParens > 0 {
		return true
	}

	return false
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
