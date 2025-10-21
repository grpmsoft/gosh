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

		// ═══════════════════════════════════════════════════════════════════════════
		// CRITICAL: Cursor Blinking in Raw Mode
		// ═══════════════════════════════════════════════════════════════════════════
		//
		// Raw mode disables automatic cursor blinking. We must explicitly enable it.
		//
		// ANSI Escape Sequences:
		//   \033[?25h         - Show cursor (DECTCEM: DEC Text Cursor Enable Mode)
		//   \033[{n} q        - DECSCUSR: Set cursor style (n = 0-6, from config)
		//
		// Cursor styles (DECSCUSR):
		//   \033[0 q or \033[ q  - Restore terminal default (usually blinking block)
		//   \033[1 q             - Blinking block █
		//   \033[2 q             - Steady block █
		//   \033[3 q             - Blinking underline _
		//   \033[4 q             - Steady underline _
		//   \033[5 q             - Blinking bar | (DEFAULT - bash/zsh/PowerShell standard)
		//   \033[6 q             - Steady bar |
		//
		// Cursor style is configurable via Config.UI.CursorStyle (default: 5 - blinking bar).
		// Users can change this in config to suit their terminal and preferences.
		//
		// PowerShell equivalent (PSReadLine/Render.cs:924-1109):
		//   _console.CursorVisible = true  (Windows Console API - blinks automatically)
		//
		// Our approach (ANSI terminals):
		//   1. Show cursor: \033[?25h
		//   2. Set cursor style from config: \033[{CursorStyle} q
		//   3. Terminal handles blinking automatically (no manual toggling needed!)
		//
		// This is executed ONCE at startup, NOT in every View() render!
		//
		// NOTE: Phoenix Input can be configured to NOT render its own cursor.
		// We use ShowCursor(false) and rely on the terminal's native cursor instead.
		// This gives us a real blinking cursor like PowerShell.
		// ═══════════════════════════════════════════════════════════════════════════

		// Show terminal cursor (Phoenix is configured not to render its own cursor via ShowCursor(false))
		fmt.Print("\033[?25h")

		// Set cursor style from config
		cursorStyle := model.Config.UI.CursorStyle

		// If blinking is disabled in config, convert to steady style
		// DECSCUSR: odd numbers = blinking, even numbers = steady
		// Example: 5 (blinking bar) → 6 (steady bar)
		if !model.Config.UI.CursorBlinking && cursorStyle%2 == 1 {
			cursorStyle++ // Convert blinking to steady
		}

		// Apply cursor style from config (default: 5 - blinking bar, PowerShell style)
		fmt.Printf("\033[%d q", cursorStyle)

		defer func() {
			// Always restore terminal state on exit
			if oldState != nil {
				// Show cursor before restoring terminal state
				fmt.Print("\033[?25h") // Show cursor
				fmt.Print("\033[0 q")  // Restore default cursor style
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

	// Set program reference in model for interactive command support (vim, ssh, claude)
	// This enables Model.execInteractiveCommand() to call p.ExecProcess()
	model.SetProgram(p) // p is already *Program[Model]

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
