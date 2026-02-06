package repl

import (
	"sync"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/grpmsoft/gosh/internal/application/execute"
	apphistory "github.com/grpmsoft/gosh/internal/application/history"
	appsession "github.com/grpmsoft/gosh/internal/application/session"
	"github.com/grpmsoft/gosh/internal/domain/config"
	"github.com/grpmsoft/gosh/internal/domain/history"
	"github.com/grpmsoft/gosh/internal/domain/session"
	"github.com/grpmsoft/gosh/internal/infrastructure/executor"
	historyInfra "github.com/grpmsoft/gosh/internal/infrastructure/history"

	"github.com/google/uuid"
	viewport "github.com/phoenix-tui/phoenix/components/viewport"
	"github.com/phoenix-tui/phoenix/style"
	tea "github.com/phoenix-tui/phoenix/tea"
	"github.com/phoenix-tui/phoenix/terminal"
)

// Global program reference for Run() compatibility
// TEMPORARY: Until Phoenix provides better mechanism for program injection
var (
	globalProgramMu sync.RWMutex
	globalProgram   *tea.Program[Model]
)

// SetGlobalProgram sets the global program reference (for Run() compatibility)
func SetGlobalProgram(p *tea.Program[Model]) {
	globalProgramMu.Lock()
	defer globalProgramMu.Unlock()
	globalProgram = p
}

// GetGlobalProgram gets the global program reference
func GetGlobalProgram() *tea.Program[Model] {
	globalProgramMu.RLock()
	defer globalProgramMu.RUnlock()
	return globalProgram
}

// Model represents REPL state (Elm Architecture).
//
// IMPORTANT: Bubbletea's MVU (Model-View-Update) pattern requires value receivers.
// for Init(), Update(), and View() methods. This means Model is passed by value (copy).
// on every update, which is the core immutability principle of the Elm architecture.
//
// Linter warnings about "hugeParam" (Model is 21KB) are expected and cannot be "fixed".
// without breaking the Bubbletea architecture. This is NOT a performance issue:
// - Bubbletea's design is optimized for this pattern.
// - Go's compiler efficiently handles struct copies.
// - The immutability benefits outweigh the copy cost.
//
//nolint:gocritic // hugeParam: Bubbletea MVU architecture requires value receivers
type Model struct {
	// Core components
	shellInput       *ShellInput         // Phoenix-based input with history navigation (single-line)
	shellTextArea    *ShellTextArea      // Phoenix-based textarea for multiline editing
	viewport         *viewport.Viewport  // Phoenix-based viewport for scrolling
	terminal         terminal.Terminal   // Phoenix Terminal (10x faster on Windows!) ⭐
	program          *tea.Program[Model] // Program reference for ExecProcess (interactive commands: vim, ssh, claude)
	sessionManager   *appsession.Manager
	executeUseCase   *execute.UseCase
	pipelineExecutor *executor.OSPipelineExecutor
	commandExecutor  *executor.OSCommandExecutor
	currentSession   *session.Session
	logger           *slog.Logger
	ctx              context.Context
	Config           *config.Config // Configuration (exported for access in main.go)

	// State
	output           []string                            // Command output (scrolls up like in terminal)
	historyNavigator *history.Navigator                  // Navigator for Up/Down arrow keys
	historyRepo      *historyInfra.FileHistoryRepository // For persistence
	addToHistoryUC   *apphistory.AddToHistoryUseCase     // Auto-save use case
	maxOutputLines   int
	width            int
	height           int
	ready            bool
	quitting         bool
	executing        bool
	lastExitCode     int
	startTime        time.Time
	gitBranch        string
	gitDirty         bool

	// Tab completion
	completions      []string
	completionIndex  int
	completionActive bool
	beforeCompletion string // Text before Tab press

	// Input state (for custom rendering)
	inputText       string
	cursorPos       int
	ghostSuggestion string // PSReadLine-style predictive suggestion from history

	// Multiline mode
	multilineMode bool // Toggle between single-line and multiline input

	// Scrolling
	autoScroll bool // Auto-scroll down on new messages

	// Help overlay
	showingHelp bool // Help overlay display flag

	// Cursor blinking
	cursorVisible bool // Cursor blink state (toggles every 500ms)

	// Styles
	styles Styles
}

// Styles contains all UI styles (PowerShell/Git Bash inspired).
// Uses Phoenix TUI Framework's style library.
type Styles struct {
	// Prompt styles
	PromptUser     style.Style
	PromptPath     style.Style
	PromptGit      style.Style
	PromptGitDirty style.Style
	PromptArrow    style.Style
	PromptError    style.Style

	// Output styles
	Output    style.Style
	OutputErr style.Style

	// Executing spinner
	Executing style.Style

	// Completion hint
	CompletionHint style.Style

	// Syntax highlighting (inline in textarea)
	SyntaxCommand style.Style // First word - command
	SyntaxOption  style.Style // --option or -o
	SyntaxArg     style.Style // Regular arguments
	SyntaxString  style.Style // "quoted strings"
}

// commandExecutedMsg message about executed command.
type commandExecutedMsg struct {
	output   string
	err      error
	exitCode int
}

// setProgramMsg is sent once after Program creation to inject program reference.
// This is necessary because MVU pattern copies Model, so program must be set via message.
type setProgramMsg struct {
	program *tea.Program[Model]
}

// NewBubbleteaREPL creates new bubbletea REPL.
func NewBubbleteaREPL(
	sessionManager *appsession.Manager,
	executeUseCase *execute.UseCase,
	logger *slog.Logger,
	ctx context.Context,
	cfg *config.Config,
) (*Model, error) {
	// Create session
	sess, err := sessionManager.CreateSession(uuid.New().String())
	if err != nil {
		return nil, err
	}

	// Create styles
	styles := makeProfessionalStyles()

	// Create viewport for scrolling (Phoenix Viewport)
	vp := viewport.New(80, 24).MouseEnabled(true)
	// Phoenix Viewport's Update() doesn't interfere with parent Model.Update(),
	// so Up/Down keys work for command history without explicit disabling

	// Create history repository and use cases
	historyFilePath := getHistoryFilePath()
	historyRepo := historyInfra.NewFileHistoryRepository(historyFilePath)

	// Load history from file into session
	loadHistoryUC := apphistory.NewLoadHistoryUseCase(historyRepo)
	if err := loadHistoryUC.Execute(sess.History()); err != nil {
		logger.Warn("Failed to load history", "error", err)
	}

	// Load .goshrc (aliases and environment)
	goshrcPath, err := config.GetDefaultGoshrcPath()
	if err != nil {
		logger.Warn("Failed to get .goshrc path", "error", err)
	} else {
		goshrcService := config.NewGoshrcService(goshrcPath)
		goshrcData, err := goshrcService.Load()
		if err != nil {
			logger.Warn("Failed to load .goshrc", "error", err)
		} else if goshrcData != nil {
			// Load aliases into session
			for name, command := range goshrcData.Aliases {
				_ = sess.SetAlias(name, command)
			}
			logger.Info("Loaded .goshrc", "aliases", len(goshrcData.Aliases), "env_vars", len(goshrcData.Environment))
		}
	}

	// Create ShellInput (Phoenix TextInput with history integration + syntax highlighting)
	shellInput := NewShellInput(80, sess.History(), applySyntaxHighlightSimple)

	// Create ShellTextArea (Phoenix TextArea with multiline support)
	shellTextArea := NewShellTextArea(80, 5, sess.History(), applySyntaxHighlightSimple)

	// Create navigator and use case for adding to history
	historyNavigator := sess.NewHistoryNavigator()
	addToHistoryUC := apphistory.NewAddToHistoryUseCase(sess.History(), historyRepo)

	// Create executors for handling commands
	pipelineExecutor := executor.NewOSPipelineExecutor(logger)
	commandExecutor := executor.NewOSCommandExecutor(logger)

	// Create Phoenix Terminal with auto-detection (Windows Console API or ANSI fallback)
	// Week 16: This gives GoSh PowerShell-level performance on Windows! ⚡
	term := terminal.New()

	m := &Model{
		shellInput:       shellInput,
		shellTextArea:    shellTextArea,
		viewport:         vp,
		terminal:         term,
		sessionManager:   sessionManager,
		executeUseCase:   executeUseCase,
		pipelineExecutor: pipelineExecutor,
		commandExecutor:  commandExecutor,
		currentSession:   sess,
		logger:           logger,
		ctx:              ctx,
		Config:           cfg,
		output:           make([]string, 0),
		historyNavigator: historyNavigator,
		historyRepo:      historyRepo,
		addToHistoryUC:   addToHistoryUC,
		maxOutputLines:   10000,
		ready:            true, // Start ready (Phoenix doesn't auto-send WindowSizeMsg yet)
		quitting:         false,
		executing:        false,
		startTime:        time.Now(),
		styles:           styles,
		completions:      []string{},
		completionIndex:  -1,
		completionActive: false,
		beforeCompletion: "",
		inputText:        "",
		cursorPos:        0,
		multilineMode:    false, // Start in single-line mode (default)
		autoScroll:       true,  // Auto-scroll down by default
		showingHelp:      false,
		cursorVisible:    true, // Start with cursor visible
		width:            80,   // Default width
		height:           24,   // Default height
	}

	// Determine Git status
	m.updateGitInfo()

	// Welcome messages (different handling for Classic vs other modes)
	welcomeLines := []string{
		"\033[1;33mGoSh\033[0m - Go Shell \033[90m(Git Bash inspired)\033[0m",
		"Press \033[1;36mF1\033[0m or \033[1;36m?\033[0m for help, \033[1;31m'exit'\033[0m to quit",
		"\033[90mSyntax: \033[1;33mcommands\033[0m yellow, \033[90moptions\033[0m gray, \033[32marguments\033[0m green\033[0m",
		"\033[90mScroll: PgUp/PgDn or Mouse Wheel\033[0m",
	}

	// Add UI mode switching hint if enabled
	if cfg.UI.AllowModeSwitching {
		welcomeLines = append(welcomeLines, "\033[90mUI Modes: Alt+1=Classic, Alt+2=Warp, Alt+3=Compact, Alt+4=Chat\033[0m")
	}

	// Handle welcome messages based on UI mode
	if cfg.UI.Mode == config.UIModeClassic {
		// Classic mode: Print welcome directly to stdout (native terminal)
		// This allows messages to remain in terminal after shell exit
		for _, line := range welcomeLines {
			fmt.Println(line)
		}
		fmt.Println() // Blank line after welcome
	} else {
		// Other modes: Add welcome to viewport buffer
		for _, line := range welcomeLines {
			m.addOutputRaw(line)
		}
		m.addOutputRaw("")

		// Initialize viewport content for non-Classic modes
		m.updateViewportContent()
	}

	return m, nil
}

// SetProgram sets the program reference for ExecProcess.
// MUST be called BEFORE tea.New() so the copy gets the reference.
func (m *Model) SetProgram(p *tea.Program[Model]) {
	m.program = p
}

// SetProgramMsg creates a message to inject program reference into Model.
// DEPRECATED: Use SetProgram() instead (called before tea.New).
func SetProgramMsg(p *tea.Program[Model]) setProgramMsg {
	return setProgramMsg{program: p}
}

// Init initializes the model (Elm Architecture).
//
//nolint:gocritic // hugeParam: Bubbletea MVU requires value receiver
func (m Model) Init() tea.Cmd {
	// No tick needed - terminal cursor blinks automatically!
	// Terminal cursor is shown and set to blinking bar style in main.go:
	//   \033[?25h - Show cursor
	//   \033[5 q  - Blinking bar style (PowerShell standard)
	//
	// Phoenix-rendered cursor is disabled via ShowCursor(false)
	// No need for manual tick/toggle - terminal handles it!
	return nil
}

// tickCmd sends a tick message every 500ms for cursor blinking.
func tickCmd() tea.Cmd {
	return tea.Tick(500 * time.Millisecond)
}

// getHistoryFilePath returns the path to the history file.
func getHistoryFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "/tmp/.gosh_history"
	}
	return filepath.Join(home, ".gosh_history")
}

// applySyntaxHighlightSimple applies simple bash syntax highlighting.
// This is used by ShellInput for real-time highlighting during typing.
//
// Highlighting rules:
// - Command (first word): Bright Yellow
// - Options (starts with -): Dark Gray
// - Arguments (other words): Green
//
// ═══════════════════════════════════════════════════════════════════════════
// CRITICAL: Whitespace Preservation (Space Key Fix!)
// ═══════════════════════════════════════════════════════════════════════════
//
// DO NOT use strings.Fields() - it REMOVES all whitespace!
// User types "echo " → Fields returns ["echo"] → spaces lost!
//
// CORRECT APPROACH: Character-by-character parsing
// - Preserve EVERY whitespace character (spaces, tabs) AS-IS
// - Only highlight words, keep whitespace unchanged
//
// This is why space key works correctly now!
// ═══════════════════════════════════════════════════════════════════════════
func applySyntaxHighlightSimple(text string) string {
	if text == "" {
		return ""
	}

	var result strings.Builder
	var currentWord strings.Builder
	wordIndex := 0
	inWord := false

	for _, ch := range text {
		if ch == ' ' || ch == '\t' {
			// Whitespace - flush current word if any
			if inWord {
				// Highlight the word
				word := currentWord.String()
				switch {
				case wordIndex == 0:
					// First word = COMMAND (YELLOW)
					result.WriteString("\033[1;33m") // Bright Yellow
					result.WriteString(word)
					result.WriteString("\033[0m")
				case strings.HasPrefix(word, "-"):
					// Option (GRAY)
					result.WriteString("\033[90m") // Dark Gray
					result.WriteString(word)
					result.WriteString("\033[0m")
				default:
					// Argument (GREEN)
					result.WriteString("\033[32m") // Green
					result.WriteString(word)
					result.WriteString("\033[0m")
				}
				currentWord.Reset()
				wordIndex++
				inWord = false
			}
			// CRITICAL: Preserve the whitespace character AS-IS!
			// This is why space key works - we don't lose the space!
			result.WriteRune(ch)
		} else {
			// Non-whitespace - accumulate word
			currentWord.WriteRune(ch)
			inWord = true
		}
	}

	// Flush last word if any
	if inWord {
		word := currentWord.String()
		switch {
		case wordIndex == 0:
			// First word = COMMAND (YELLOW)
			result.WriteString("\033[1;33m")
			result.WriteString(word)
			result.WriteString("\033[0m")
		case strings.HasPrefix(word, "-"):
			// Option (GRAY)
			result.WriteString("\033[90m")
			result.WriteString(word)
			result.WriteString("\033[0m")
		default:
			// Argument (GREEN)
			result.WriteString("\033[32m")
			result.WriteString(word)
			result.WriteString("\033[0m")
		}
	}

	return result.String()
}
