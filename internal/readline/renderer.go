package readline

import (
	"strings"

	"github.com/unilibs/uniwidth"
)

// Renderer handles coordinate transformations and rendering.
// Based on PowerShell 7's PSReadLine rendering logic.
type Renderer struct {
	console  Console
	initialX int // Initial cursor X position (after prompt)
	initialY int // Initial cursor Y position (top of buffer)

	// Current render state
	currentBuffer string // Current input buffer
	currentCursor int    // Cursor offset in buffer
}

// NewRenderer creates a new readline renderer.
func NewRenderer(console Console, initialX, initialY int) *Renderer {
	return &Renderer{
		console:  console,
		initialX: initialX,
		initialY: initialY,
	}
}

// ConvertOffsetToPoint converts text offset to screen coordinates.
//
// This is the core function from PSReadLine/Render.cs:1364-1419.
// It accounts for:
// - Unicode character widths (using Phoenix RuneWidth)
// - Line wrapping at buffer width
// - Continuation prompts (not implemented yet)
//
// Example:
//
//	buffer = "echo hello мир"
//	offset = 10 (position before 'м')
//	initialX = 5, initialY = 0
//	bufferWidth = 80
//	→ Point{X: 16, Y: 0}
func (r *Renderer) ConvertOffsetToPoint(offset int, buffer string) Point {
	x := r.initialX
	y := r.initialY

	bufferWidth := r.console.BufferWidth()

	// Continuation prompt length (PSReadLine has this, we'll add later)
	// For now, continuation prompt is empty
	continuationPromptLength := 0

	byteIndex := 0
	for byteIndex < offset && byteIndex < len(buffer) {
		// Decode rune at current byte position
		c, size := DecodeRuneAt(buffer, byteIndex)

		if c == '\n' {
			y++
			x = continuationPromptLength
			byteIndex += size
		} else {
			// Use uniwidth for Unicode-aware width calculation
			cellWidth := uniwidth.RuneWidth(c)
			x += cellWidth

			// Wrap to next line if we exceed buffer width
			if x >= bufferWidth {
				// If character exactly fit (x == bufferWidth), cursor moves to column 0
				// If character didn't fit (x > bufferWidth), it wraps entirely to next line
				wasExactFit := (x == bufferWidth)
				if wasExactFit {
					x = 0
				} else {
					x = cellWidth
				}

				// Always increment y after wrapping, unless next is newline
				nextByteIndex := byteIndex + size
				shouldIncrementY := true

				if nextByteIndex < offset && nextByteIndex < len(buffer) {
					nextChar, _ := DecodeRuneAt(buffer, nextByteIndex)
					if nextChar == '\n' && !wasExactFit {
						// Don't increment if next is newline (newline will handle it)
						shouldIncrementY = false
					}
				}

				if shouldIncrementY {
					y++
				}
			}
			byteIndex += size
		}
	}

	// If next character exists and isn't newline, check if it's wider than remaining space
	if len(buffer) > offset {
		c, _ := DecodeRuneAt(buffer, offset)
		if c != '\n' {
			cellWidth := uniwidth.RuneWidth(c)
			if x+cellWidth > bufferWidth {
				// Character is wider than remaining space, so it wraps to next line
				x = 0
				y++
			}
		}
	} else {
		// At end of buffer - if cursor is exactly at buffer width, move to next line
		if x == bufferWidth {
			x = 0
			y++
		}
	}

	return Point{X: x, Y: y}
}

// ConvertPointToOffset converts screen coordinates to text offset.
//
// This is the inverse of ConvertOffsetToPoint.
// Based on PSReadLine/Render.cs:1421-1475.
//
// Returns -1 if the point is out of range.
func (r *Renderer) ConvertPointToOffset(point Point, buffer string) int {
	x := r.initialX
	y := r.initialY

	bufferWidth := r.console.BufferWidth()
	continuationPromptLength := 0

	byteIndex := 0
	for byteIndex < len(buffer) {
		// If we're on the correct line and column, return the offset
		if point.Y == y && point.X <= x {
			return byteIndex
		}

		// Decode rune at current byte position
		c, size := DecodeRuneAt(buffer, byteIndex)

		if c == '\n' {
			// If we're about to move off the correct line, return current offset
			if point.Y == y {
				return byteIndex
			}
			y++
			x = continuationPromptLength
			byteIndex += size
		} else {
			cellWidth := uniwidth.RuneWidth(c)
			x += cellWidth

			// Wrap handling
			if x >= bufferWidth {
				if x == bufferWidth {
					x = 0
				} else {
					x = cellWidth
				}

				nextByteIndex := byteIndex + size
				if nextByteIndex < len(buffer) {
					nextChar, _ := DecodeRuneAt(buffer, nextByteIndex)
					if nextChar != '\n' {
						y++
					} else if x != 0 {
						y++
					}
				} else if x != 0 {
					y++
				}
			}
			byteIndex += size
		}
	}

	// Return last offset if point.Y matches, otherwise -1
	if point.Y == y {
		return len(buffer)
	}
	return -1
}

// SkipAnsiSequences skips ANSI escape sequences when counting visible characters.
//
// ANSI sequences have format: ESC [ ... m
// This is used when processing rendered text with color codes.
//
// Returns the next index after the sequence, or current index if not a sequence.
func SkipAnsiSequences(text string, i int) int {
	// Check for ESC [ sequence
	if i < len(text) && text[i] == '\033' && i+1 < len(text) && text[i+1] == '[' {
		i += 2 // Skip ESC and [

		// Skip until we find 'm' (end of ANSI color sequence)
		for i < len(text) && text[i] != 'm' {
			i++
		}

		// Skip the 'm'
		if i < len(text) {
			i++
		}
	}

	return i
}

// CountVisibleRunes counts visible runes in text, skipping ANSI escape codes.
//
// This is useful for calculating actual display width of syntax-highlighted text.
//
// Example:
//
//	text = "\033[32mecho\033[0m hello"  // "echo hello" with green "echo"
//	CountVisibleRunes(text) == 10        // 4 (echo) + 1 (space) + 5 (hello)
func CountVisibleRunes(text string) int {
	count := 0

	for i := 0; i < len(text); {
		// Skip ANSI escape sequences
		if text[i] == '\033' && i+1 < len(text) && text[i+1] == '[' {
			i = SkipAnsiSequences(text, i)
			continue
		}

		// Count visible rune
		count++
		i++
	}

	return count
}

// RecomputeInitialCoords recomputes initial coordinates after terminal resize.
//
// Based on PSReadLine/Render.cs:1189-1312.
// This is called when the terminal size changes to adjust the rendering.
//
// Simplified version - full PSReadLine version handles many edge cases.
func (r *Renderer) RecomputeInitialCoords(currentCursor int, buffer string) {
	// Update console size
	if sc, ok := r.console.(*SystemConsole); ok {
		_ = sc.UpdateSize()
	}

	// Recompute X from buffer width (handle prompt wrap)
	bufferWidth := r.console.BufferWidth()
	r.initialX %= bufferWidth

	// Recompute Y from cursor position
	// Calculate cursor position assuming initialY = 0
	r.initialY = 0
	pt := r.ConvertOffsetToPoint(currentCursor, buffer)

	// Adjust initialY based on actual cursor position
	// (In real implementation, we'd query console cursor position)
	// For now, we assume cursor is at pt
	actualCursorTop := pt.Y // Placeholder - would query console
	r.initialY = actualCursorTop - pt.Y
}

// Render renders the buffer with syntax highlighting and cursor positioning.
//
// Based on PSReadLine's ReallyRender (Render.cs:905-1120).
//
// This implements differential rendering:
// - Only redraws changed portions
// - Clears old content as needed
// - Positions cursor correctly
//
// Parameters:
//   - buffer: The input text to render
//   - cursorOffset: Cursor position in the buffer
//   - highlightedText: The buffer with ANSI syntax highlighting applied
//
// Returns the final cursor position.
func (r *Renderer) Render(buffer string, cursorOffset int, highlightedText string) Point {
	bufferHeight := r.console.BufferHeight()

	// Hide cursor during rendering (PowerShell approach)
	r.console.SetCursorVisible(false)
	defer r.console.SetCursorVisible(true)

	// Move to initial position
	r.console.SetCursorPosition(r.initialX, r.initialY)

	// Simple rendering for now - just write the highlighted text
	// TODO: Implement differential rendering like PSReadLine
	r.console.Write(highlightedText)

	// Calculate cursor position
	point := r.ConvertOffsetToPoint(cursorOffset, buffer)

	// Handle cursor at buffer edge (would scroll)
	if point.Y >= bufferHeight {
		// Scroll up by writing newline
		r.console.Write("\n")
		r.initialY--
		point.Y--
	}

	// Position cursor
	r.console.SetCursorPosition(point.X, point.Y)

	// Update state
	r.currentBuffer = buffer
	r.currentCursor = cursorOffset

	return point
}

// Clear clears the current line.
// Uses ANSI escape code to erase from cursor to end of line.
func (r *Renderer) Clear() {
	r.console.Write("\r\033[2K") // CR + clear entire line
}

// Spaces returns a string of n spaces.
// Used for clearing portions of the screen.
func Spaces(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat(" ", n)
}

// DecodeRuneAt decodes a single rune starting at the given byte index.
// Returns the rune and its byte length.
// If the index is at the end of the string, returns (0, 0).
func DecodeRuneAt(s string, byteIndex int) (rune, int) {
	if byteIndex >= len(s) {
		return 0, 0
	}

	// Get first byte
	b := s[byteIndex]

	// Fast path: ASCII (single byte)
	if b < 0x80 {
		return rune(b), 1
	}

	// Multi-byte UTF-8
	// Decode using built-in UTF-8 decoding
	r := rune(0)
	size := 0

	// 2-byte sequence: 110xxxxx 10xxxxxx
	if b&0xE0 == 0xC0 {
		if byteIndex+1 < len(s) {
			r = rune(b&0x1F)<<6 | rune(s[byteIndex+1]&0x3F)
			size = 2
		}
	}

	// 3-byte sequence: 1110xxxx 10xxxxxx 10xxxxxx
	if b&0xF0 == 0xE0 {
		if byteIndex+2 < len(s) {
			r = rune(b&0x0F)<<12 | rune(s[byteIndex+1]&0x3F)<<6 | rune(s[byteIndex+2]&0x3F)
			size = 3
		}
	}

	// 4-byte sequence: 11110xxx 10xxxxxx 10xxxxxx 10xxxxxx
	if b&0xF8 == 0xF0 {
		if byteIndex+3 < len(s) {
			r = rune(b&0x07)<<18 | rune(s[byteIndex+1]&0x3F)<<12 | rune(s[byteIndex+2]&0x3F)<<6 | rune(s[byteIndex+3]&0x3F)
			size = 4
		}
	}

	if size == 0 {
		// Invalid UTF-8, return replacement character
		return 0xFFFD, 1
	}

	return r, size
}
