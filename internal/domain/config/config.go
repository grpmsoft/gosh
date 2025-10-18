// Package config provides domain configuration models for shell behavior and UI modes.
package config

// UIMode defines the terminal display mode.
type UIMode string

const (
	// UIModeClassic - classic mode (bash/pwsh): output at top, prompt comes after.
	UIModeClassic UIMode = "classic"

	// UIModeWarp - modern mode (Warp): prompt always at top, output below with viewport.
	UIModeWarp UIMode = "warp"

	// UIModeCompact - compact mode without separators.
	UIModeCompact UIMode = "compact"

	// UIModeChat - chat mode (Telegram/ChatGPT): input fixed at bottom, history scrolls.
	// Ideal for AI agents and conversational interfaces.
	UIModeChat UIMode = "chat"
)

// UICategory defines UI mode category (determines technical approach).
type UICategory string

const (
	// UICategoryTerminal - terminal modes (classic, warp, compact).
	UICategoryTerminal UICategory = "terminal"

	// UICategoryChat - chat modes (chat) - require viewport.
	UICategoryChat UICategory = "chat"
)

// GetUICategory returns the category for a mode.
func GetUICategory(mode UIMode) UICategory {
	if mode == UIModeChat {
		return UICategoryChat
	}
	return UICategoryTerminal
}

// Config represents GoSh configuration.
type Config struct {
	// UI settings
	UI UIConfig `json:"ui"`

	// Shell settings
	Shell ShellConfig `json:"shell"`

	// History settings
	History HistoryConfig `json:"history"`
}

// UIConfig interface settings.
type UIConfig struct {
	// Mode - display mode (classic, warp, compact, chat)
	Mode UIMode `json:"mode"`

	// Theme - color theme (monokai, dracula, solarized)
	Theme string `json:"theme"`

	// ShowGitStatus - show git status in prompt
	ShowGitStatus bool `json:"show_git_status"`

	// SyntaxHighlight - enable syntax highlighting
	SyntaxHighlight bool `json:"syntax_highlight"`

	// CompletionEnabled - enable auto-completion
	CompletionEnabled bool `json:"completion_enabled"`

	// AllowModeSwitching - allow mode switching with hotkeys
	AllowModeSwitching bool `json:"allow_mode_switching"`

	// OutputSeparator - separator printed after command output in Classic mode
	// Default: "\n" (empty line like bash). Can be "" (no separator) or any string.
	OutputSeparator string `json:"output_separator"`

	// CursorBlinking - enable cursor blinking animation (like PowerShell)
	// true: cursor blinks every 500ms (default)
	// false: cursor is always visible (static)
	CursorBlinking bool `json:"cursor_blinking"`

	// CursorStyle - cursor style using DECSCUSR codes (ANSI \033[{n} q)
	// DECSCUSR - DEC Set Cursor Style:
	//   0 - Restore terminal default (usually blinking block)
	//   1 - Blinking block █
	//   2 - Steady block █
	//   3 - Blinking underline _
	//   4 - Steady underline _
	//   5 - Blinking bar | (DEFAULT - bash/zsh/PowerShell standard)
	//   6 - Steady bar |
	//
	// Note: Some terminals (MSYS/Git Bash) may not support all cursor styles in raw mode.
	// If blinking doesn't work, try changing to a different code or update terminal.
	CursorStyle int `json:"cursor_style"`
}

// ShellConfig shell settings.
type ShellConfig struct {
	// DefaultShell - default shell for executing commands
	DefaultShell string `json:"default_shell"`

	// Env - additional environment variables
	Env map[string]string `json:"env"`
}

// HistoryConfig command history settings.
type HistoryConfig struct {
	// MaxLines - maximum lines in history
	MaxLines int `json:"max_lines"`

	// SaveToFile - save history to file
	SaveToFile bool `json:"save_to_file"`

	// FilePath - path to history file
	FilePath string `json:"file_path"`
}

// DefaultConfig returns default configuration.
func DefaultConfig() *Config {
	return &Config{
		UI: UIConfig{
			Mode:               UIModeClassic, // Default classic mode (like bash/pwsh)
			Theme:              "monokai",
			ShowGitStatus:      true,
			SyntaxHighlight:    true,
			CompletionEnabled:  true,
			AllowModeSwitching: true, // Allow mode switching via F1-F4
			OutputSeparator:    "\n", // Empty line after command output (bash-style)
			CursorBlinking:     true, // Blinking cursor like PowerShell (configurable)
			CursorStyle:        5,    // Blinking bar cursor (DECSCUSR code 5 - PowerShell/bash standard)
		},
		Shell: ShellConfig{
			DefaultShell: "sh",
			Env:          make(map[string]string),
		},
		History: HistoryConfig{
			MaxLines:   10000,
			SaveToFile: true,
			FilePath:   "~/.gosh_history",
		},
	}
}
