package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/grpmsoft/gosh/internal/application/execute"
	"github.com/phoenix-tui/phoenix/tea/api"
)

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

	// Run (without AltScreen - using native terminal scrolling)
	p := api.New(model)

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
