package repl

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/grpmsoft/gosh/internal/application/execute"
	apphistory "github.com/grpmsoft/gosh/internal/application/history"
	appsession "github.com/grpmsoft/gosh/internal/application/session"
	"github.com/grpmsoft/gosh/internal/domain/config"
	"github.com/grpmsoft/gosh/internal/domain/history"
	"github.com/grpmsoft/gosh/internal/domain/session"
	"github.com/grpmsoft/gosh/internal/infrastructure/executor"
	historyInfra "github.com/grpmsoft/gosh/internal/infrastructure/history"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/phoenix-tui/phoenix/tea/api"
)

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
	shellInput       *ShellInput    // Phoenix-based input with history navigation
	viewport         viewport.Model
	sessionManager   *appsession.Manager
	executeUseCase   *execute.UseCase
	pipelineExecutor *executor.OSPipelineExecutor
	commandExecutor  *executor.OSCommandExecutor
	currentSession   *session.Session
	logger           *slog.Logger
	ctx              context.Context
	config           *config.Config // Configuration

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

	// Spinner for execution
	executingSpinner spinner.Model

	// Tab completion
	completions      []string
	completionIndex  int
	completionActive bool
	beforeCompletion string // Text before Tab press

	// Input state (for custom rendering)
	inputText string
	cursorPos int

	// Scrolling
	autoScroll bool // Auto-scroll down on new messages

	// Help overlay
	showingHelp bool // Help overlay display flag

	// Styles
	styles Styles
}

// Styles contains all UI styles (PowerShell/Git Bash inspired).
type Styles struct {
	// Prompt styles
	PromptUser     lipgloss.Style
	PromptPath     lipgloss.Style
	PromptGit      lipgloss.Style
	PromptGitDirty lipgloss.Style
	PromptArrow    lipgloss.Style
	PromptError    lipgloss.Style

	// Output styles
	Output    lipgloss.Style
	OutputErr lipgloss.Style

	// Executing spinner
	Executing lipgloss.Style

	// Completion hint
	CompletionHint lipgloss.Style

	// Syntax highlighting (inline in textarea)
	SyntaxCommand lipgloss.Style // First word - command
	SyntaxOption  lipgloss.Style // --option or -o
	SyntaxArg     lipgloss.Style // Regular arguments
	SyntaxString  lipgloss.Style // "quoted strings"
}

// commandExecutedMsg message about executed command.
type commandExecutedMsg struct {
	output   string
	err      error
	exitCode int
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

	// Create spinner
	execSpinner := spinner.New()
	execSpinner.Spinner = spinner.Dot
	execSpinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("12")) // Blue

	// Create viewport for scrolling
	vp := viewport.New(80, 24)
	vp.MouseWheelEnabled = true
	// Disable default up/down keys (needed for command history)
	vp.KeyMap.Up.SetEnabled(false)
	vp.KeyMap.Down.SetEnabled(false)
	// Keep PageUp/PageDown enabled

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

	// Create ShellInput (Phoenix TextInput with history integration)
	shellInput := NewShellInput(80, sess.History())

	// Create navigator and use case for adding to history
	historyNavigator := sess.NewHistoryNavigator()
	addToHistoryUC := apphistory.NewAddToHistoryUseCase(sess.History(), historyRepo)

	// Create executors for handling commands
	pipelineExecutor := executor.NewOSPipelineExecutor(logger)
	commandExecutor := executor.NewOSCommandExecutor(logger)

	m := &Model{
		shellInput:       shellInput,
		viewport:         vp,
		sessionManager:   sessionManager,
		executeUseCase:   executeUseCase,
		pipelineExecutor: pipelineExecutor,
		commandExecutor:  commandExecutor,
		currentSession:   sess,
		logger:           logger,
		ctx:              ctx,
		config:           cfg,
		output:           make([]string, 0),
		historyNavigator: historyNavigator,
		historyRepo:      historyRepo,
		addToHistoryUC:   addToHistoryUC,
		maxOutputLines:   10000,
		ready:            false,
		quitting:         false,
		executing:        false,
		startTime:        time.Now(),
		styles:           styles,
		executingSpinner: execSpinner,
		completions:      []string{},
		completionIndex:  -1,
		completionActive: false,
		beforeCompletion: "",
		inputText:        "",
		cursorPos:        0,
		autoScroll:       true, // Auto-scroll down by default
		showingHelp:      false,
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

// Init initializes the model (Elm Architecture).
//
//nolint:gocritic // hugeParam: Bubbletea MVU requires value receiver
func (m Model) Init() api.Cmd {
	// ShellInput handles cursor blinking internally, no need for explicit Blink command
	return nil
}

// getHistoryFilePath returns the path to the history file.
func getHistoryFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "/tmp/.gosh_history"
	}
	return filepath.Join(home, ".gosh_history")
}
