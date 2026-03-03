// Package readline provides PowerShell-style readline functionality for Classic mode.
//
// This package implements coordinate transformations and cursor management
// based on PowerShell 7's PSReadLine implementation.
//
// Key features:
// - Offset ↔ (X,Y) coordinate transformations
// - Unicode width handling (via Phoenix)
// - Terminal resize support
// - Differential rendering
// - Native terminal scrolling (NOT AltScreen!)
package readline

import "fmt"

// Point represents a console cursor position.
// Based on PowerShell's PSReadLine Point struct.
type Point struct {
	X int // Column (0-based)
	Y int // Row (0-based, relative to initial position)
}

func (p Point) String() string {
	return fmt.Sprintf("%d,%d", p.X, p.Y)
}
