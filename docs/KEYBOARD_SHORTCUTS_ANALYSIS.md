# 🎹 Keyboard Shortcuts Analysis for GoSh

**Date**: 2025-10-12
**Version**: 0.1.0-beta.2
**Status**: Recommendations for consideration

---

## 📜 Historical Context

### F1 for Help - The Standard Since 1987

The **F1 key for help** originated from the **IBM Common User Access (CUA)** guidelines published in 1987. This became the de facto standard across:

- **DOS applications** (Norton Commander, WordPerfect)
- **Windows applications** (universal help key)
- **Linux terminals** (many TUI applications)
- **IBM mainframes** (PF1 key for help)

**Why F1?**
- Easy to reach (top-left of keyboard)
- Dedicated function key (not used for text input)
- Consistent across applications
- 38 years of user muscle memory

---

## 🔍 Modern TUI Best Practices (2025)

### Research Sources Analyzed

1. **lazygit** - Popular Git TUI
   - Uses `?` for keybindings display
   - Context-sensitive help
   - Quote: "Display the context-sensitive keybindings with ?"

2. **k9s** - Kubernetes TUI
   - Keyboard-driven interface
   - `?` for help/shortcuts
   - Extensive keyboard navigation

3. **btop** - System monitor TUI
   - Help menu with detailed shortcuts
   - Function keys for mode switching
   - `h` or `?` for help

4. **clig.dev** - Modern CLI Guidelines
   - Recommends `--help`, `-h`, `help` command
   - Multiple ways to get help
   - Context-sensitive assistance

5. **GDB TUI** - Debugger interface
   - C-x 1/2 for layout switching
   - Arrow keys for navigation
   - Help command for assistance

### Key Findings

✅ **Multiple Help Mechanisms**: Modern apps offer several ways:
- `?` key (interactive, instant)
- `--help` flag (CLI standard)
- `help` command (shell convention)
- F1 key (legacy/traditional)

✅ **Keyboard-First Design**: Power users prefer:
- Single-key shortcuts (no Ctrl/Alt required)
- Vim-style navigation where appropriate
- Mnemonic keys (h=help, q=quit, etc.)

✅ **Context-Sensitive Help**: Best practice:
- Different help in different modes
- Show only relevant shortcuts
- Visual help overlays

✅ **Discoverability**: Critical for new users:
- Show "Press ? for help" in prompt/status
- Visual hints for common actions
- Progressive disclosure of advanced shortcuts

---

## 🧐 Current GoSh Implementation

### Existing Keyboard Shortcuts

From `internal/interfaces/repl/bubbletea_repl.go:382`:

| Key | Action | Category |
|-----|--------|----------|
| **F1** | Switch to Classic mode | UI Mode |
| **F2** | Switch to Warp mode | UI Mode |
| **F3** | Switch to Compact mode | UI Mode |
| **F4** | Switch to Chat mode | UI Mode |
| **Tab** | Auto-completion | Input |
| **↑/↓** | History navigation | Input |
| **Enter** | Execute command | Input |
| **Alt+Enter** | Multi-line input | Input |
| **Ctrl+C** | Exit (or interrupt) | Control |
| **Ctrl+D** | Exit if empty input | Control |
| **Ctrl+L** | Clear screen | Control |
| **PgUp/PgDn** | Scroll output | Navigation |
| **Mouse Wheel** | Scroll output | Navigation |

### Help System

**Current Implementation**:
- `help` command displays built-in help
- Shows keyboard shortcuts list
- Lists built-in commands
- No visual overlay

**Invocation**:
```bash
gosh> help
```

**Output** (from `bubbletea_repl.go:1253`):
```
Keyboard shortcuts:
  Tab          - Auto-complete
  ↑/↓          - History
  PgUp/PgDn    - Scroll output
  Mouse Wheel  - Scroll output
  Alt+Enter    - New line (multiline)
  Ctrl+L       - Clear screen
  Ctrl+C/D     - Exit
```

---

## 🎯 Analysis & Recommendations

### Issue 1: F1-F4 for UI Modes (Non-Standard)

**Problem**:
- Conflicts with traditional F1=Help convention (38 years of muscle memory)
- Function keys typically used for less frequent actions
- UI mode switching might not be a common operation

**Modern Examples**:
- **btop**: F2 for UI setup menu (infrequent)
- **lazygit**: No function keys for mode switching
- **vim**: `:set` commands for UI changes (not hotkeys)

**Recommendation**: Consider alternative approaches:

**Option A - Ctrl+F5-F8 for UI Modes** ⭐ **RECOMMENDED & ACCEPTED**
```
F1        → Help/Keybindings display (restore traditional convention)
F2-F8     → Reserved for future frequent features (7 keys!)
?         → Quick help overlay (add new)
ESC       → Close help overlay / Cancel operations

Ctrl+F5-F8 → UI Modes (infrequent operation, modifier appropriate)
  Ctrl+F5  → Classic mode
  Ctrl+F6  → Warp mode
  Ctrl+F7  → Compact mode
  Ctrl+F8  → Chat mode

:mode [name] → Command-based mode switching (fallback)
```

**Pros**:
- ✅ Restores F1=Help standard (38-year convention)
- ✅ **F2-F8 freed for frequent operations** (7 valuable function keys!)
- ✅ ESC reserved for closing overlays (standard pattern)
- ✅ Adds modern `?` key help
- ✅ **No conflicts with standard shortcuts** (Ctrl+F4=Close Tab avoided!)
- ✅ Ctrl modifier appropriate for infrequent mode switching
- ✅ Protection against accidental mode changes

**Cons**:
- Breaking change for beta.2 users (but breaking changes still allowed!)
- Need to update documentation
- Requires two keys (but mode switching is infrequent)

**Why Ctrl+F5-F8 (not Ctrl+F1-F4)**:
- ❌ Ctrl+F4 = Close Tab (browsers, IDE) - strong muscle memory conflict
- ❌ Ctrl+F1-F3 sometimes used in IDEs
- ✅ Ctrl+F5-F8 are rarely used, safe from conflicts
- ✅ Allows F2-F8 for future frequent operations

**Option B - Keep Current, Add ?**
```
F1-4 → Keep for UI modes
?    → Add help overlay
help → Keep command
```

**Pros**:
- No breaking changes
- Adds modern help method

**Cons**:
- F1 still conflicts with traditional expectation
- Confusing for new users expecting F1=Help

### Issue 2: No Visual Help Overlay

**Problem**:
- `help` command requires typing and execution
- Output gets mixed with shell history
- Not immediately visible to new users

**Modern Pattern** (lazygit, k9s):
- Press `?` → Instant help overlay appears
- Shows context-sensitive shortcuts
- Press `?` or `Esc` to close
- Doesn't interfere with work

**Recommendation**: Add `?` key for visual help overlay

**Implementation Ideas**:
1. **Modal Help Window** (like lazygit):
   ```
   ┌─────────────────────────────────────┐
   │ GoSh Keyboard Shortcuts             │
   ├─────────────────────────────────────┤
   │ Navigation:                         │
   │   ↑/↓      - Command history        │
   │   Tab      - Auto-complete          │
   │   PgUp/Dn  - Scroll output          │
   │                                     │
   │ Input:                              │
   │   Enter    - Execute command        │
   │   Alt+Enter - Multi-line            │
   │   Ctrl+L   - Clear screen           │
   │                                     │
   │ UI Modes:                           │
   │   Ctrl+F5  - Classic mode           │
   │   Ctrl+F6  - Warp mode              │
   │   Ctrl+F7  - Compact mode           │
   │   Ctrl+F8  - Chat mode              │
   │                                     │
   │ Help:                               │
   │   F1/?     - This help              │
   │   help     - Detailed help          │
   │                                     │
   │ Press ESC to close                  │
   └─────────────────────────────────────┘
   ```

2. **Context-Sensitive Help**:
   - Different help in different UI modes
   - Show mode-specific shortcuts
   - Highlight most useful keys

### Issue 3: Discoverability

**Problem**:
- New users don't know about `help` command
- No visual hints in UI
- Classic mode has minimal prompts

**Modern Solutions**:
1. **Status Line Hints** (like bottom bar in many TUIs):
   ```
   gosh> _
   [?=Help] [↑↓=History] [Tab=Complete] [Ctrl+C=Exit]
   ```

2. **First-Run Tutorial**:
   ```
   Welcome to GoSh v0.1.0-beta.2!

   Quick tips:
   - Press ? for keyboard shortcuts
   - Type 'help' for detailed help
   - Try Tab for auto-completion

   Press any key to continue...
   ```

3. **Prompt Integration** (Warp mode already has visual prompt):
   - Add subtle `?` hint to prompt
   - Show in status area

### Issue 4: Other Keyboard Shortcuts to Consider

**Missing Modern Conventions**:

1. **Ctrl+R - Reverse Search** (bash/zsh standard)
   - Currently: Not implemented
   - Recommendation: Add fuzzy history search (v0.2.0 roadmap)

2. **Ctrl+W - Delete Word** (readline standard)
   - Currently: Not implemented
   - Recommendation: Add word-based editing

3. **Ctrl+A/E - Line Start/End** (readline/emacs standard)
   - Currently: Bubbletea textinput handles this (need to verify)
   - Recommendation: Ensure readline compatibility

4. **Ctrl+K/U - Kill Line** (readline standard)
   - Currently: Not implemented
   - Recommendation: Add line editing shortcuts

5. **Esc or q - Quick Exit from Help**
   - Currently: Not applicable (no help overlay)
   - Recommendation: Add when implementing ? overlay

---

## 📋 Proposed Changes Summary

### Phase 1: Beta.3 (Immediate - Breaking Changes Allowed)

**Priority: HIGH** - Restore F1 convention

1. **Repurpose F1 for Help**:
   - F1 → Show help overlay (restore traditional convention)
   - Add `?` key → Same help overlay (modern TUI pattern)
   - Keep `help` command for detailed text help

2. **UI Mode Switching Changes**:
   - Ctrl+F5 → Classic mode (infrequent operation)
   - Ctrl+F6 → Warp mode (was F2)
   - Ctrl+F7 → Compact mode (was F3)
   - Ctrl+F8 → Chat mode (was F4)
   - Ctrl modifier prevents accidental switching
   - **F2-F8 freed** for future frequent features (7 keys!)
   - Add `:mode [name]` command (fallback method)

3. **Add Visual Help Overlay**:
   - Press F1 or ? → Modal help window appears
   - Show context-sensitive shortcuts
   - Press ESC to close (standard pattern)
   - Bubbletea lipgloss styling

4. **Update Documentation**:
   - README.md keyboard shortcuts section
   - `help` command output
   - User guide (docs/USER_GUIDE.md when created)

### Phase 2: v0.2.0 (Post-Stable)

**Priority: MEDIUM** - Enhanced keyboard experience

1. **Readline Compatibility**:
   - Ctrl+A/E - Line start/end
   - Ctrl+K/U - Kill line forward/backward
   - Ctrl+W - Delete word backward
   - Ctrl+Y - Yank (paste killed text)

2. **Fuzzy History Search**:
   - Ctrl+R - Reverse interactive search
   - Visual search UI with filtering
   - Already in v0.2.0 roadmap

3. **Discoverability Enhancements**:
   - First-run welcome screen
   - Status bar with hints
   - Tooltips in UI modes

4. **Vi Mode** (Optional):
   - Vi-style editing mode (if requested by community)
   - Toggle with `set -o vi` (bash compatibility)

---

## 🎨 Implementation Notes

### Technical Considerations

**Bubbletea Key Handling** (from `bubbletea_repl.go:382`):
```go
case key.Matches(msg, m.keyMap.Enter):
    return m.handleEnter()

case key.Matches(msg, m.keyMap.F1):
    return m.switchUIMode(UIClassic)  // Current implementation

// Proposed change:
case key.Matches(msg, m.keyMap.F1), key.Matches(msg, m.keyMap.Help):
    return m.showHelpOverlay()  // New implementation
```

**Help Overlay with Lipgloss**:
```go
// Pseudo-code
func (m *Model) renderHelpOverlay() string {
    helpStyle := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("62")).
        Padding(1, 2).
        Width(50)

    content := "GoSh Keyboard Shortcuts\n\n" +
               "Navigation:\n" +
               "  ↑/↓      - Command history\n" +
               "  Tab      - Auto-complete\n" +
               "...\n" +
               "Press ? or Esc to close"

    return helpStyle.Render(content)
}
```

**Context-Sensitive Help**:
```go
func (m *Model) getContextHelp() []string {
    switch m.uiMode {
    case UIClassic:
        return []string{"Basic mode shortcuts...", ...}
    case UIWarp:
        return []string{"Warp mode features...", ...}
    case UICompact:
        return []string{"Compact mode shortcuts...", ...}
    case UIChat:
        return []string{"Chat mode interactions...", ...}
    }
}
```

### Testing Considerations

**Key Binding Tests** (add to test suite):
```go
func TestHelpOverlay(t *testing.T) {
    // Test F1 shows help
    // Test ? shows help
    // Test Esc closes help
    // Test help content is correct
}

func TestUIModeSwitch(t *testing.T) {
    // Test Ctrl+F5-F8 mode switching (Classic/Warp/Compact/Chat)
    // Test :mode command
    // Test mode persistence
    // Verify F2-F8 reserved for future use (7 keys)
}
```

**User Acceptance Testing**:
- Verify F1 feels natural for help
- Check ? key doesn't conflict with shell patterns
- Ensure overlay doesn't break layouts
- Test on all 3 platforms (Linux/macOS/Windows)

---

## 💡 Recommendations Summary

### ✅ Strongly Recommend (Beta.3)

1. **F1 → Help** - Restore 38-year tradition
2. **? → Help** - Follow modern TUI pattern (lazygit, k9s)
3. **ESC → Close overlays** - Standard cancel/close pattern
4. **Ctrl+F5-F8 → UI Modes** - Infrequent operation (Classic/Warp/Compact/Chat)
5. **F2-F8 freed** - 7 keys for future frequent features
6. **Implement Visual Help Overlay** - Better UX than command
7. **Add :mode command** - Fallback for keyboard-less scenarios

### 🤔 Consider (v0.2.0)

1. **Readline Compatibility** - Ctrl+A/E/K/U/W for power users
2. **First-Run Tutorial** - Help new users discover features
3. **Status Bar Hints** - Constant discoverability
4. **Ctrl+R Fuzzy Search** - Already in roadmap

### ❌ Not Recommended

1. **Remove help command** - Keep for scripts and accessibility
2. **Overload ? key** - Don't use for shell glob patterns (conflict)
3. **Too many function keys** - F1-F4 is enough

---

## 📊 Comparison with Other Shells

### Traditional Shells

**Bash**:
- F1: Not used (terminal-dependent)
- Ctrl+R: Reverse search
- Ctrl+A/E/K/U: Readline editing
- `help` command: Shows built-in commands

**Zsh**:
- Similar to bash
- More advanced completion
- Bindkey customization

**Fish**:
- F1: Shows command help (context-sensitive!)
- Alt+H: Man page for current command
- ? not used (part of shell syntax)

### Modern TUI Shells

**Nu Shell**:
- No special help key
- `help` command
- Rich documentation system

**Oil Shell**:
- Traditional readline bindings
- `help` command
- Focus on scripting

### GoSh Current Position

**Strengths**:
- 4 UI modes (unique!)
- Good basic shortcuts
- Multi-platform

**Opportunities**:
- Align F1 with tradition
- Add modern ? help
- Visual help system
- Better discoverability

---

## 🎯 Final Recommendation

### For v0.1.0-beta.3 (Next Release)

**Goal**: Restore F1 convention while adding modern help pattern

**Changes**:
1. **F1 → Help** (restore 38-year tradition!)
2. **? → Help** (modern TUI pattern)
3. **ESC → Close overlays** (cancel operations)
4. **Ctrl+F5-F8 → UI Modes** (infrequent operation, modifier appropriate)
   - Ctrl+F5: Classic mode
   - Ctrl+F6: Warp mode (was F2)
   - Ctrl+F7: Compact mode (was F3)
   - Ctrl+F8: Chat mode (was F4)
5. **F2-F8 → Reserved** for future frequent features (7 keys freed!)
6. **Add :mode [name]** command for fallback switching
7. **Implement help overlay** with lipgloss styling
8. **Update all documentation** with new shortcuts

**Benefits**:
- ✅ Aligns with 38 years of F1=Help convention
- ✅ Follows modern TUI best practices (?, visual help)
- ✅ **F2-F8 freed for frequent operations** (7 valuable function keys!)
- ✅ ESC properly reserved for closing overlays
- ✅ **No shortcut conflicts** (avoids Ctrl+F4=Close Tab)
- ✅ Ctrl modifier prevents accidental mode switching
- ✅ Appropriate complexity for infrequent operation
- ✅ Better user experience (instant help)
- ✅ Improved discoverability
- ✅ Still in beta - breaking changes acceptable
- ✅ Positions GoSh as modern yet familiar

**Migration Path**:
- Document breaking change in CHANGELOG.md
- Update README.md prominently
- Add note in beta.3 release announcement
- Beta.3 → RC.1 → v0.1.0 timeline allows user feedback

---

**Next Steps**:
1. Review this analysis with project team
2. Decide on Phase 1 implementation
3. Create GitHub issue for tracking
4. Implement changes for beta.3
5. Update tests and documentation
6. Gather community feedback

---

*Created: 2025-10-12*
*Author: AI-assisted analysis*
*Status: Awaiting review*
