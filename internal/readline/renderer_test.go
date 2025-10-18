package readline

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// MockConsole is a mock console for testing.
type MockConsole struct {
	width  int
	height int
}

func NewMockConsole(width, height int) *MockConsole {
	return &MockConsole{width: width, height: height}
}

func (m *MockConsole) BufferWidth() int               { return m.width }
func (m *MockConsole) BufferHeight() int              { return m.height }
func (m *MockConsole) CursorLeft() int                { return 0 }
func (m *MockConsole) CursorTop() int                 { return 0 }
func (m *MockConsole) SetCursorPosition(left, top int) {}
func (m *MockConsole) Write(text string)             {}
func (m *MockConsole) SetCursorVisible(visible bool) {}

func TestConvertOffsetToPoint_Simple(t *testing.T) {
	console := NewMockConsole(80, 24)
	renderer := NewRenderer(console, 5, 0) // initialX=5 (after prompt), initialY=0

	tests := []struct {
		name   string
		buffer string
		offset int
		want   Point
	}{
		{
			name:   "start of buffer",
			buffer: "echo hello",
			offset: 0,
			want:   Point{X: 5, Y: 0}, // At initial position
		},
		{
			name:   "middle of buffer",
			buffer: "echo hello",
			offset: 5, // After "echo "
			want:   Point{X: 10, Y: 0}, // 5 (initial) + 5 (chars)
		},
		{
			name:   "end of buffer",
			buffer: "echo hello",
			offset: 10,
			want:   Point{X: 15, Y: 0}, // 5 (initial) + 10 (chars)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderer.ConvertOffsetToPoint(tt.offset, tt.buffer)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConvertOffsetToPoint_Wrapping(t *testing.T) {
	console := NewMockConsole(20, 24) // Narrow terminal
	renderer := NewRenderer(console, 5, 0)

	tests := []struct {
		name   string
		buffer string
		offset int
		want   Point
	}{
		{
			name:   "exact fit at buffer width",
			buffer: "123456789012345", // 15 chars + 5 initial = 20 (exact fit)
			offset: 15,
			want:   Point{X: 0, Y: 1}, // Wraps to next line, column 0
		},
		{
			name:   "overflow buffer width",
			buffer: "1234567890123456", // 16 chars + 5 initial = 21 (overflow)
			offset: 16,
			want:   Point{X: 1, Y: 1}, // Wraps to next line, column 1
		},
		{
			name:   "middle of second line",
			buffer: "12345678901234567890", // 20 chars
			offset: 20,
			want:   Point{X: 5, Y: 1}, // Second line, 5 chars in
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderer.ConvertOffsetToPoint(tt.offset, tt.buffer)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConvertOffsetToPoint_Unicode(t *testing.T) {
	console := NewMockConsole(80, 24)
	renderer := NewRenderer(console, 5, 0)

	tests := []struct {
		name   string
		buffer string
		offset int
		want   Point
	}{
		{
			name:   "ascii only",
			buffer: "echo hello",
			offset: 5,
			want:   Point{X: 10, Y: 0},
		},
		{
			name:   "russian (2-byte UTF-8, 1 cell width)",
			buffer: "echo привет", // "привет" = 6 runes, 12 bytes, 6 cells
			offset: 5,              // After "echo "
			want:   Point{X: 10, Y: 0},
		},
		{
			name:   "chinese (3-byte UTF-8, 2 cell width)",
			buffer: "echo 你好",  // "你好" = 2 runes, 6 bytes, 4 cells (wide chars)
			offset: 11,          // After "echo 你好" (5 + 3 + 3 bytes)
			want:   Point{X: 14, Y: 0}, // 5 (initial) + 5 ("echo ") + 4 (2 wide chars)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderer.ConvertOffsetToPoint(tt.offset, tt.buffer)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConvertOffsetToPoint_Newlines(t *testing.T) {
	console := NewMockConsole(80, 24)
	renderer := NewRenderer(console, 5, 0)

	tests := []struct {
		name   string
		buffer string
		offset int
		want   Point
	}{
		{
			name:   "before newline",
			buffer: "line1\nline2",
			offset: 5, // At '\n'
			want:   Point{X: 10, Y: 0}, // 5 (initial) + 5 ("line1")
		},
		{
			name:   "after newline",
			buffer: "line1\nline2",
			offset: 6, // After '\n', at 'l'
			want:   Point{X: 0, Y: 1}, // Second line, column 0 (no continuation prompt yet)
		},
		{
			name:   "middle of second line",
			buffer: "line1\nline2",
			offset: 9, // After "line1\nlin"
			want:   Point{X: 3, Y: 1}, // Second line, column 3
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderer.ConvertOffsetToPoint(tt.offset, tt.buffer)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConvertPointToOffset(t *testing.T) {
	console := NewMockConsole(80, 24)
	renderer := NewRenderer(console, 5, 0)

	tests := []struct {
		name   string
		buffer string
		point  Point
		want   int
	}{
		{
			name:   "start of buffer",
			buffer: "echo hello",
			point:  Point{X: 5, Y: 0},
			want:   0,
		},
		{
			name:   "middle of buffer",
			buffer: "echo hello",
			point:  Point{X: 10, Y: 0},
			want:   5,
		},
		{
			name:   "end of buffer",
			buffer: "echo hello",
			point:  Point{X: 15, Y: 0},
			want:   10,
		},
		{
			name:   "second line",
			buffer: "line1\nline2",
			point:  Point{X: 3, Y: 1},
			want:   9, // After "line1\nlin"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderer.ConvertPointToOffset(tt.point, tt.buffer)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConvertOffsetToPoint_RoundTrip(t *testing.T) {
	// Test that ConvertOffsetToPoint and ConvertPointToOffset are inverses
	console := NewMockConsole(80, 24)
	renderer := NewRenderer(console, 5, 0)

	buffers := []string{
		"echo hello",
		"echo hello world this is a test",
		"line1\nline2\nline3",
		"echo 你好 world",
		"a\nb\nc\nd\ne",
	}

	for _, buffer := range buffers {
		// Iterate over valid rune boundaries, not all byte positions
		byteIndex := 0
		for byteIndex <= len(buffer) {
			// Convert offset → point → offset
			point := renderer.ConvertOffsetToPoint(byteIndex, buffer)
			gotOffset := renderer.ConvertPointToOffset(point, buffer)

			assert.Equal(t, byteIndex, gotOffset,
				"Round trip failed for buffer=%q offset=%d: point=%v gotOffset=%d",
				buffer, byteIndex, point, gotOffset)

			// Move to next rune boundary
			if byteIndex < len(buffer) {
				_, size := DecodeRuneAt(buffer, byteIndex)
				byteIndex += size
			} else {
				break
			}
		}
	}
}

func TestSkipAnsiSequences(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		i     int
		want  int
	}{
		{
			name: "simple color code",
			text: "\033[32mhello",
			i:    0,
			want: 5, // After "\033[32m"
		},
		{
			name: "not a sequence",
			text: "hello",
			i:    0,
			want: 0, // No change
		},
		{
			name: "reset code",
			text: "text\033[0mmore",
			i:    4,
			want: 8, // After "\033[0m"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SkipAnsiSequences(tt.text, tt.i)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCountVisibleRunes(t *testing.T) {
	tests := []struct {
		name string
		text string
		want int
	}{
		{
			name: "plain text",
			text: "hello",
			want: 5,
		},
		{
			name: "with color",
			text: "\033[32mhello\033[0m",
			want: 5, // "hello" without ANSI codes
		},
		{
			name: "multiple colors",
			text: "\033[1;33mecho\033[0m \033[32mworld\033[0m",
			want: 10, // "echo world"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CountVisibleRunes(tt.text)
			assert.Equal(t, tt.want, got)
		})
	}
}
