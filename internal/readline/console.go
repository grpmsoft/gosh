package readline

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

// Console provides low-level console operations.
// Based on PowerShell's IConsole interface.
type Console interface {
	// BufferWidth returns the width of the console buffer in characters.
	BufferWidth() int

	// BufferHeight returns the height of the console buffer in characters.
	BufferHeight() int

	// CursorLeft returns the current cursor column (0-based).
	CursorLeft() int

	// CursorTop returns the current cursor row (0-based).
	CursorTop() int

	// SetCursorPosition sets the cursor to the specified column and row.
	SetCursorPosition(left, top int)

	// Write writes text to the console.
	Write(text string)

	// CursorVisible gets or sets cursor visibility.
	SetCursorVisible(visible bool)
}

// SystemConsole implements Console using os.Stdout and terminal control codes.
// This is for native terminal (NOT AltScreen!) - like PowerShell's behavior.
type SystemConsole struct {
	fd     int
	width  int
	height int
}

// NewSystemConsole creates a new system console.
func NewSystemConsole() (*SystemConsole, error) {
	fd := int(os.Stdout.Fd())

	width, height, err := term.GetSize(fd)
	if err != nil {
		return nil, fmt.Errorf("failed to get terminal size: %w", err)
	}

	return &SystemConsole{
		fd:     fd,
		width:  width,
		height: height,
	}, nil
}

func (c *SystemConsole) BufferWidth() int {
	return c.width
}

func (c *SystemConsole) BufferHeight() int {
	return c.height
}

func (c *SystemConsole) CursorLeft() int {
	// In PowerShell, this queries console directly
	// For us, we track this in Renderer state
	// This is a placeholder - actual tracking done in Renderer
	return 0
}

func (c *SystemConsole) CursorTop() int {
	// In PowerShell, this queries console directly
	// For us, we track this in Renderer state
	// This is a placeholder - actual tracking done in Renderer
	return 0
}

func (c *SystemConsole) SetCursorPosition(left, top int) {
	// ANSI escape code: CSI {row};{col} H
	// Note: ANSI uses 1-based, we use 0-based
	fmt.Printf("\033[%d;%dH", top+1, left+1)
}

func (c *SystemConsole) Write(text string) {
	fmt.Print(text)
}

func (c *SystemConsole) SetCursorVisible(visible bool) {
	if visible {
		fmt.Print("\033[?25h") // DECTCEM - show cursor
	} else {
		fmt.Print("\033[?25l") // DECTCEM - hide cursor
	}
}

// UpdateSize updates the cached terminal size.
// Call this after detecting a resize (SIGWINCH).
func (c *SystemConsole) UpdateSize() error {
	width, height, err := term.GetSize(c.fd)
	if err != nil {
		return fmt.Errorf("failed to get terminal size: %w", err)
	}

	c.width = width
	c.height = height
	return nil
}
