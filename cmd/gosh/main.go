package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/grpmsoft/gosh/internal/application/execute"
	"github.com/grpmsoft/gosh/internal/interfaces/repl"
	"github.com/phoenix-tui/phoenix/tea"
)

// autoFlushWriter wraps an io.Writer and flushes after each Write.
// Needed for raw mode where stdout is buffered.
type autoFlushWriter struct {
	w *bufio.Writer
}

func newAutoFlushWriter(w io.Writer) *autoFlushWriter {
	return &autoFlushWriter{w: bufio.NewWriter(w)}
}

func (a *autoFlushWriter) Write(p []byte) (n int, err error) {
	n, err = a.w.Write(p)
	if err != nil {
		return n, err
	}
	// Always flush after write in raw mode
	return n, a.w.Flush()
}

func main() {
	// Parse command line flags
	commandFlag := flag.String("c", "", "Execute command and exit (non-interactive mode)")
	modeFlag := flag.String("mode", "", "UI mode: classic, warp, compact, chat")
	flag.Parse()

	// Setup
	logger := setupLogger()
	ctx := context.Background()

	// Non-interactive mode: -c "command"
	if *commandFlag != "" {
		exitCode := executeNonInteractive(ctx, logger, *commandFlag)
		os.Exit(exitCode)
	}

	// Validate --mode flag
	modeOverride := strings.ToLower(strings.TrimSpace(*modeFlag))
	if modeOverride != "" {
		switch modeOverride {
		case "classic", "warp", "compact", "chat":
			// valid
		default:
			fmt.Fprintf(os.Stderr, "Unknown mode: %s (available: classic, warp, compact, chat)\n", *modeFlag)
			os.Exit(1)
		}
	}

	// Auto-flush stdout after each write.
	// In raw mode stdout is buffered and Phoenix doesn't flush.
	stdout := newAutoFlushWriter(os.Stdout)

	// Bootstrap REPL
	model, err := bootstrapREPL(logger, ctx, modeOverride)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create REPL: %v\n", err)
		os.Exit(1)
	}

	// Phoenix TUI options — NEVER use WithAltScreen.
	// Alt screen is managed manually by the Model (enter/exit via ANSI sequences).
	// This allows runtime mode switching without restarting the Program.
	var opts []tea.Option[repl.Model]
	opts = append(opts,
		tea.WithInput[repl.Model](os.Stdin),
		tea.WithOutput[repl.Model](stdout),
	)

	p := tea.New(*model, opts...)

	// Set global program reference for ExecProcess compatibility
	repl.SetGlobalProgram(p)

	// Run BLOCKS main thread (single program, no restart loop)
	if err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}

// executeNonInteractive executes a command in non-interactive mode (-c flag).
func executeNonInteractive(ctx context.Context, logger *slog.Logger, commandLine string) int {
	// Create session and use case for execution
	sessionManager, executeUseCase := bootstrapNonInteractive(logger)

	// Create temporary session
	sess, err := sessionManager.CreateSession("non-interactive")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create session: %v\n", err)
		return 1
	}

	// Execute command
	resp, err := executeUseCase.Execute(
		ctx,
		execute.CommandRequest{
			CommandLine: commandLine,
			SessionID:   sess.ID(),
		},
		sess,
	)

	// Output result
	if resp != nil {
		if resp.Stdout != "" {
			fmt.Print(resp.Stdout)
			// Add newline if it's missing
			if !strings.HasSuffix(resp.Stdout, "\n") {
				fmt.Println()
			}
		}
		if resp.Stderr != "" {
			fmt.Fprint(os.Stderr, resp.Stderr)
			if !strings.HasSuffix(resp.Stderr, "\n") {
				fmt.Fprintln(os.Stderr)
			}
		}
	}

	// Handle error
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	// Return command exit code
	if resp != nil {
		return int(resp.ExitCode)
	}

	return 0
}
