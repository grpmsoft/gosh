package repl

import (
	"github.com/grpmsoft/gosh/internal/application/execute"
	"github.com/grpmsoft/gosh/internal/interfaces/parser"

	tea "github.com/charmbracelet/bubbletea"
)

// isBuiltinCommand checks if command is builtin (cd, export, unset).
// These commands must execute synchronously in shell process.
func (m *Model) isBuiltinCommand(cmdName string) bool {
	builtinCommands := map[string]bool{
		"cd":      true,
		"export":  true,
		"unset":   true,
		"pwd":     true,
		"echo":    true,
		"env":     true,
		"alias":   true,
		"unalias": true,
		"type":    true,
		"jobs":    true,
		"fg":      true,
		"bg":      true,
	}

	return builtinCommands[cmdName]
}

// execBuiltinCommand executes builtin command synchronously via executeUseCase.
func (m *Model) execBuiltinCommand(commandLine string) tea.Cmd {
	return func() tea.Msg {
		// Parse command
		cmd, _, err := parser.ParseCommandLine(commandLine)
		if err != nil {
			return commandExecutedMsg{
				err:      err,
				exitCode: 1,
			}
		}

		if cmd == nil {
			return commandExecutedMsg{
				output:   "",
				exitCode: 0,
			}
		}

		// Execute via executeUseCase which correctly delegates to BuiltinExecutor
		resp, err := m.executeUseCase.Execute(
			m.ctx,
			execute.ExecuteCommandRequest{
				CommandLine: commandLine,
				SessionID:   m.currentSession.ID(),
			},
			m.currentSession,
		)

		if err != nil {
			return commandExecutedMsg{
				err:      err,
				exitCode: 1,
			}
		}

		// Return result
		output := ""
		exitCode := 0

		if resp != nil {
			output = resp.Stdout
			if resp.Stderr != "" {
				if output != "" {
					output += "\n"
				}
				output += resp.Stderr
			}
			exitCode = int(resp.ExitCode)
		}

		return commandExecutedMsg{
			output:   output,
			err:      err,
			exitCode: exitCode,
		}
	}
}
