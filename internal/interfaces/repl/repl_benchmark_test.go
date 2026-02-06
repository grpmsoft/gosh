package repl

import (
	"os"
	"testing"

	"github.com/grpmsoft/gosh/internal/domain/history"
	"github.com/grpmsoft/gosh/internal/domain/session"
	"github.com/grpmsoft/gosh/internal/domain/shared"
	"github.com/phoenix-tui/phoenix/tea"
)

// BenchmarkShellInputUpdate measures ShellInput Update performance
func BenchmarkShellInputUpdate(b *testing.B) {
	hist := history.NewHistory(history.DefaultConfig())
	input := NewShellInput(80, hist, applySyntaxHighlightSimple)

	msg := tea.KeyMsg{Type: tea.KeyRune, Rune: 'a'}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		input.Update(msg)
	}
}

// BenchmarkShellInputView measures ShellInput View rendering (with syntax highlighting)
func BenchmarkShellInputView(b *testing.B) {
	hist := history.NewHistory(history.DefaultConfig())
	input := NewShellInput(80, hist, applySyntaxHighlightSimple)
	input.SetValue("ls -la | grep test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		input.View()
	}
}

// BenchmarkSyntaxHighlighting measures syntax highlighting performance
func BenchmarkSyntaxHighlighting(b *testing.B) {
	testCases := []struct {
		name  string
		input string
	}{
		{"Simple command", "ls -la"},
		{"Pipeline", "ls -la | grep test | sort"},
		{"Quoted string", `echo "hello world"`},
		{"Variables", "echo $PATH ${HOME}"},
		{"Complex", `git commit -m "feat: add feature" && git push origin main`},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				applySyntaxHighlightSimple(tc.input)
			}
		})
	}
}

// BenchmarkHistoryNavigation measures history navigation performance
func BenchmarkHistoryNavigation(b *testing.B) {
	hist := history.NewHistory(history.DefaultConfig())

	// Add 100 commands to history
	for i := 0; i < 100; i++ {
		hist.Add("command" + string(rune('0'+i%10)))
	}

	input := NewShellInput(80, hist, applySyntaxHighlightSimple)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		input.Update(tea.KeyMsg{Type: tea.KeyUp})
		input.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
}

// BenchmarkSessionOperations measures session operations performance
func BenchmarkSessionOperations(b *testing.B) {
	env := make(shared.Environment)

	b.Run("ChangeDirectory", func(b *testing.B) {
		sess, _ := session.NewSession("bench-session", os.TempDir(), env)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sess.ChangeDirectory(os.TempDir())
		}
	})

	b.Run("SetVariable", func(b *testing.B) {
		sess, _ := session.NewSession("bench-session", os.TempDir(), env)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sess.SetVariable("TEST_VAR", "test_value")
		}
	})

	b.Run("GetVariable", func(b *testing.B) {
		sess, _ := session.NewSession("bench-session", os.TempDir(), env)
		sess.SetVariable("TEST_VAR", "test_value")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sess.GetVariable("TEST_VAR")
		}
	})
}