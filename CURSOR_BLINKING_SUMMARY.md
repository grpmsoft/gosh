# Cursor Blinking Fix - Summary

**Date**: 2025-01-18
**Version**: v0.0.00-alpha
**Status**: FIXED

---

## Problem

System cursor in GoSh shell was **visible but NOT blinking** (steady/static).

---

## Root Cause

**Raw terminal mode disables automatic cursor blinking.**

When `term.MakeRaw()` is called:
- Line buffering: DISABLED
- Character echo: DISABLED
- Signal handling: DISABLED
- **Cursor blinking: DISABLED** ⭐

The sequence `\033[?25h` (show cursor) makes cursor **visible** but **not blinking** in raw mode.

---

## Solution

Use **DECSCUSR (DEC Set Cursor Style)** sequence to explicitly enable blinking:

```go
fmt.Print("\033[?25h")  // Show cursor (DECTCEM)
fmt.Print("\033[5 q")   // Set blinking bar cursor (DECSCUSR)
```

This makes cursor:
1. Visible (`\033[?25h`)
2. Bar-shaped (`5` in DECSCUSR)
3. Blinking (odd numbers blink in DECSCUSR)

---

## Implementation

**File**: `cmd/gosh/main.go` (lines 73-118)

**Before**:
```go
// Show cursor (ANSI: \033[?25h)
fmt.Print("\033[?25h")  // Cursor visible but NOT blinking
```

**After**:
```go
// Show cursor and enable blinking
fmt.Print("\033[?25h")  // Show cursor
fmt.Print("\033[5 q")   // Set blinking bar cursor style
```

**On Exit**:
```go
defer func() {
    if oldState != nil {
        fmt.Print("\033[0 q")  // Restore default cursor style
        _ = term.Restore(int(os.Stdin.Fd()), oldState)
    }
}()
```

---

## DECSCUSR Quick Reference

Syntax: `\033[{n} q`

| Code | Style | Blink | Usage |
|------|-------|-------|-------|
| `0` | Default | Usually yes | Restore terminal default |
| `1` | Block █ | Yes | Traditional |
| `2` | Block █ | No | Steady block |
| `3` | Underline _ | Yes | Vi replace mode |
| `4` | Underline _ | No | Steady underline |
| `5` | **Bar \|** | **Yes** | **Shells (bash, zsh, PowerShell)** ⭐ |
| `6` | Bar \| | No | Steady bar |

**GoSh uses**: `\033[5 q` (blinking bar) - matches modern shell conventions.

---

## Testing

### Build and Run
```bash
cd /d/projects/grpmsoft/gosh
go build ./cmd/gosh
./gosh.exe
```

### Expected Result
- Cursor appears as vertical bar `|`
- Cursor **blinks** on/off automatically
- Matches behavior of bash, zsh, PowerShell

### Previous Result
- Cursor appeared as vertical bar `|`
- Cursor was **static** (no blinking)

---

## Why This Matters

1. **User Experience**: Blinking cursor is standard shell behavior
2. **Visibility**: Easier to locate cursor position while typing
3. **Expectations**: Users expect cursor to blink (from bash, zsh, PowerShell)
4. **Professionalism**: Matches production-quality shell behavior

---

## Files Modified

1. `cmd/gosh/main.go` (lines 73-118)
   - Added `\033[5 q` for blinking bar cursor
   - Added `\033[0 q` on exit to restore default
   - Updated documentation comments

2. `internal/interfaces/repl/shell_input.go` (lines 124-165)
   - Updated documentation explaining DECSCUSR
   - Clarified PowerShell vs ANSI terminal differences

3. `docs/dev/CURSOR_BLINKING_FIX.md` (NEW)
   - Comprehensive technical documentation
   - DECSCUSR reference
   - PowerShell vs GoSh comparison

4. `test_cursor_blink.sh` (NEW)
   - Interactive test script
   - Demonstrates all DECSCUSR cursor styles

---

## PowerShell vs GoSh

| Aspect | PowerShell | GoSh |
|--------|------------|------|
| API | Windows Console API | ANSI escape sequences |
| Cursor visibility | `CursorVisible = true` | `\033[?25h` |
| Cursor blinking | Automatic (hardware) | `\033[5 q` (DECSCUSR) |
| Cross-platform | Windows only | Unix, Linux, macOS, Git Bash |

---

## Future Enhancements

When implementing **Vi mode**, use different cursor styles:
- **Insert mode**: `\033[5 q` (blinking bar) - current
- **Normal mode**: `\033[1 q` (blinking block)
- **Replace mode**: `\033[3 q` (blinking underline)

---

## References

- [ANSI Escape Codes (Wikipedia)](https://en.wikipedia.org/wiki/ANSI_escape_code)
- [XTerm Control Sequences](https://invisible-island.net/xterm/ctlseqs/ctlseqs.html)
- [VT100.net - DECSCUSR](https://vt100.net/docs/vt510-rm/DECSCUSR.html)
- PowerShell PSReadLine: `Render.cs:924-1109`

---

**Tests**: All tests pass
**Impact**: Visual UX improvement - cursor now blinks correctly
**Compatibility**: All major terminals (Windows Terminal, Git Bash, xterm, etc.)
