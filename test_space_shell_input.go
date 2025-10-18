package main

import (
	"fmt"
	"os"

	"github.com/grpmsoft/gosh/internal/domain/history"
	"github.com/grpmsoft/gosh/internal/interfaces/repl"
	tea "github.com/phoenix-tui/phoenix/tea/api"
	"golang.org/x/term"
)

type model struct {
	input *repl.ShellInput
	hist  *history.History
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
		if msg.Type == tea.KeyEnter {
			fmt.Printf("\nEntered: '%s'\n", m.input.Value())
			m.input.Reset()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return "Test ShellInput (press Ctrl+C to quit):\n" + m.input.View()
}

func main() {
	// Enable raw mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to set raw mode: %v\n", err)
		os.Exit(1)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Show cursor and set blinking bar style
	fmt.Print("\033[?25h") // Show cursor
	fmt.Print("\033[5 q")  // Blinking bar

	defer func() {
		fmt.Print("\033[?25h") // Show cursor
		fmt.Print("\033[0 q")  // Restore default cursor style
	}()

	hist := history.New(100)
	highlight := func(text string) string {
		// Simple test: just return text as-is (no highlighting)
		return text
	}
	input := repl.NewShellInput(40, hist, highlight)

	p := tea.NewProgram(model{input: input, hist: hist})
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
