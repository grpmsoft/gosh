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
	"github.com/phoenix-tui/phoenix/tea/api"
	"golang.org/x/term"
)

// autoFlushWriter wraps an io.Writer and flushes after each Write.
// Needed for raw mode where stdout is buffered.
type autoFlushWriter struct {
	w *bufio.Writer
}

func newAutoFlushWriter(w io.Writer) *autoFlushWriter {
	return &autoFlushWriter{
		w: bufio.NewWriter(w),
	}
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
	flag.Parse()

	// Setup
	logger := setupLogger()
	ctx := context.Background()

	// Non-interactive mode: -c "command"
	if *commandFlag != "" {
		exitCode := executeNonInteractive(ctx, logger, *commandFlag)
		os.Exit(exitCode)
	}

	// Interactive mode: Bootstrap REPL
	model, err := bootstrapREPL(logger, ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create REPL: %v\n", err)
		os.Exit(1)
	}

	// TEMPORARY FIX: Enable raw mode for terminal (only if stdin is a TTY)
	// Phoenix doesn't set up raw mode yet (planned for later weeks)
	// In raw mode, terminal doesn't echo typed characters - only our View() renders them
	var oldState *term.State
	if term.IsTerminal(int(os.Stdin.Fd())) {
		oldState, err = term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to set raw mode: %v\n", err)
			os.Exit(1)
		}

		// CRITICAL: Hide system cursor (we render our own cursor in View())
		// ANSI: \033[?25l = hide cursor, \033[?25h = show cursor
		fmt.Print("\033[?25l") // Hide system cursor

		defer func() {
			// Always restore terminal state on exit
			fmt.Print("\033[?25h") // Show system cursor
			if oldState != nil {
				_ = term.Restore(int(os.Stdin.Fd()), oldState)
			}
		}()
	}

	// TEMPORARY FIX: Auto-flush stdout after each write
	// In raw mode, stdout is buffered and Phoenix doesn't flush
	// This wrapper flushes automatically after every Write()
	stdout := newAutoFlushWriter(os.Stdout)

	// Run (without AltScreen - using native terminal scrolling)
	// Phoenix tea/api requires value type for MVU pattern
	// Enable mouse support for viewport scrolling + auto-flush output
	p := api.New(*model,
		api.WithMouseAllMotion[repl.Model](),
		api.WithOutput[repl.Model](stdout),
	)

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
