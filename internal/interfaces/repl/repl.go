package repl

import (
	"context"
	"errors"
	"fmt"
	"github.com/grpmsoft/gosh/internal/application/execute"
	appsession "github.com/grpmsoft/gosh/internal/application/session"
	"github.com/grpmsoft/gosh/internal/domain/session"
	"github.com/grpmsoft/gosh/internal/interfaces/parser"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strings"

	"github.com/chzyer/readline"
	"github.com/google/uuid"
)

// REPL represents interactive shell interface
type REPL struct {
	sessionManager *appsession.SessionManager
	executeUseCase *execute.ExecuteCommandUseCase
	currentSession *session.Session
	rl             *readline.Instance
	logger         *slog.Logger
	promptTemplate string
	running        bool
}

// NewREPL creates a new REPL
func NewREPL(
	sessionManager *appsession.SessionManager,
	executeUseCase *execute.ExecuteCommandUseCase,
	logger *slog.Logger,
) (*REPL, error) {
	// Create session
	sess, err := sessionManager.CreateSession(uuid.New().String())
	if err != nil {
		return nil, err
	}

	// Create readline instance
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          buildPrompt(sess),
		HistoryFile:     getHistoryFilePath(),
		AutoComplete:    completer,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		return nil, err
	}

	return &REPL{
		sessionManager: sessionManager,
		executeUseCase: executeUseCase,
		currentSession: sess,
		rl:             rl,
		logger:         logger,
		promptTemplate: "gosh",
		running:        false,
	}, nil
}

// Run starts the REPL
func (r *REPL) Run(ctx context.Context) error {
	r.running = true
	defer func() { _ = r.Close() }()

	r.printWelcome()

	for r.running {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Check for completed background jobs and display notifications
			r.checkCompletedJobs()

			// Update prompt
			r.rl.SetPrompt(buildPrompt(r.currentSession))

			// Read line
			line, err := r.rl.Readline()
			if err != nil {
				if errors.Is(err, readline.ErrInterrupt) {
					if len(line) == 0 {
						// Ctrl+C on empty line - exit
						return nil
					} else {
						// Ctrl+C on non-empty line - cancel current input
						continue
					}
				} else if errors.Is(err, io.EOF) {
					// Ctrl+D - exit
					return nil
				}
				return err
			}

			// Execute command
			if err := r.executeLine(ctx, line); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}
		}
	}

	return nil
}

// executeLine executes the entered line
func (r *REPL) executeLine(ctx context.Context, line string) error {
	line = strings.TrimSpace(line)

	// Skip empty lines and comments
	if line == "" || strings.HasPrefix(line, "#") {
		return nil
	}

	// Parse command
	cmd, pipe, err := parser.ParseCommandLine(line)
	if err != nil {
		return err
	}

	// Create execution request
	req := execute.ExecuteCommandRequest{
		CommandLine: line,
		SessionID:   r.currentSession.ID(),
	}

	// Execute through use case
	// Temporarily using direct parsing as use case is not fully integrated yet
	if pipe != nil {
		// For now just indicate that pipeline is not supported
		fmt.Println("Pipeline execution not yet fully integrated")
		return nil
	}

	if cmd != nil {
		// Simple command execution
		// TODO: Integrate with use case properly
		fmt.Printf("Command parsed: %s with %d args\n", cmd.Name(), len(cmd.Args()))
	}

	// Temporary stub
	_, err = r.executeUseCase.Execute(ctx, req, r.currentSession)
	return err
}

// checkCompletedJobs checks for completed background jobs and displays notifications
func (r *REPL) checkCompletedJobs() {
	jobManager := r.currentSession.JobManager()
	if jobManager == nil {
		return
	}

	jobs := jobManager.ListJobs()
	for _, job := range jobs {
		proc := job.Process()

		// Sync job state with process state
		if job.IsRunning() && proc.IsCompleted() {
			// Process completed, update job state
			if proc.ExitCode() == 0 {
				_ = job.Complete()
			} else {
				_ = job.Fail()
			}

			// Display notification
			status := "Done"
			if proc.ExitCode() != 0 {
				status = fmt.Sprintf("Exit %d", proc.ExitCode())
			}

			fmt.Printf("\n[%d] %s    %s\n",
				job.JobNumber(),
				status,
				job.Command().FullCommand())
		}
	}

	// Remove finished jobs
	jobManager.RemoveFinishedJobs()
}

// Close closes the REPL
func (r *REPL) Close() error {
	r.running = false
	if r.rl != nil {
		_ = r.rl.Close()
	}
	if r.currentSession != nil {
		_ = r.sessionManager.CloseSession(r.currentSession.ID())
	}
	return nil
}

// printWelcome prints welcome message
func (r *REPL) printWelcome() {
	fmt.Println("Welcome to gosh - Go Shell")
	fmt.Printf("Version: 0.1.0 | Go: %s | OS: %s/%s\n",
		runtime.Version(),
		runtime.GOOS,
		runtime.GOARCH,
	)
	fmt.Println("Type 'help' for available commands or 'exit' to quit")
	fmt.Println()
}

// buildPrompt builds the prompt string
func buildPrompt(sess *session.Session) string {
	// Simplified prompt: gosh:workdir$
	workDir := sess.WorkingDirectory()

	// Get only current directory name
	parts := strings.Split(workDir, string(os.PathSeparator))
	currentDir := parts[len(parts)-1]
	if currentDir == "" && len(parts) > 1 {
		currentDir = parts[len(parts)-2]
	}

	return fmt.Sprintf("gosh:%s$ ", currentDir)
}

// getHistoryFilePath returns the path to history file
func getHistoryFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "/tmp/.gosh_history"
	}
	return home + string(os.PathSeparator) + ".gosh_history"
}

// completer - command auto-completion
var completer = readline.NewPrefixCompleter(
	readline.PcItem("cd"),
	readline.PcItem("pwd"),
	readline.PcItem("echo"),
	readline.PcItem("exit"),
	readline.PcItem("export"),
	readline.PcItem("unset"),
	readline.PcItem("env"),
	readline.PcItem("type"),
	readline.PcItem("help"),
	readline.PcItem("jobs"),
	readline.PcItem("fg"),
	readline.PcItem("bg"),
	readline.PcItem("ls"),
	readline.PcItem("cat"),
	readline.PcItem("grep"),
	readline.PcItem("find"),
)
