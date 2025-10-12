# GoSh UI Modes

GoSh supports multiple interface display modes. Choose the one that suits you best!

## UI Modes

### 1. Classic Mode (default) 🐚

**Like bash/pwsh/zsh** - traditional terminal:

```
GoSh - Go Shell
Type 'help' for commands

Andy@DESKTOP /d/projects $ ls -la
total 588
drwxr-xr-x 1 Andy 197121   0 Oct 11 14:22 .
drwxr-xr-x 1 Andy 197121   0 Oct 10 20:15 ..
drwxr-xr-x 1 Andy 197121   0 Oct 11 03:59 .idea

Andy@DESKTOP /d/projects $ █
```

**Features:**
- Command output flows sequentially from top to bottom
- Prompt appears AFTER the previous command's output
- Scrolling is native terminal (Ctrl+Shift+↑/↓ or mouse wheel)
- **The most familiar mode for everyone**

**Config:**
```json
{
  "ui": {
    "mode": "classic"
  }
}
```

---

### 2. Warp Mode 🚀

**Like Warp terminal** - modern approach:

```
┌────────────────────────────────────────┐
│ Andy@DESKTOP /d/projects $ ls -la█     │ ← Prompt ALWAYS at top
├────────────────────────────────────────┤
│ total 588                              │
│ drwxr-xr-x 1 Andy 197121   0 Oct 11   │ ← Output below
│ drwxr-xr-x 1 Andy 197121   0 Oct 10   │   (with viewport scrolling)
│ ...                                    │
│                                        │
└────────────────────────────────────────┘
```

**Features:**
- Input prompt **always pinned at the top**
- Command output displayed below in scrollable viewport
- Convenient to see what you're typing even with long output
- Scrolling via PgUp/PgDn or Mouse Wheel

**Config:**
```json
{
  "ui": {
    "mode": "warp"
  }
}
```

---

### 3. Compact Mode ⚡

**Minimalist** - without unnecessary elements:

```
$ ls -la█
total 588
drwxr-xr-x 1 Andy 197121   0 Oct 11 14:22 .
$ █
```

**Features:**
- Minimal prompt (just `$`)
- No git status or decorations
- Maximum space for command output
- For those who love simplicity

**Config:**
```json
{
  "ui": {
    "mode": "compact"
  }
}
```

---

## How to Choose a Mode?

### 1. Create config file

Copy the example:
```bash
cp .goshrc.example ~/.goshrc
```

### 2. Edit

Open `~/.goshrc` and change `mode`:

```json
{
  "ui": {
    "mode": "classic"  // or "warp" or "compact"
  }
}
```

### 3. Restart GoSh

Changes apply on next startup.

---

## Which Mode to Choose?

| Mode | Best For |
|------|----------|
| **classic** | bash/zsh/pwsh users - familiar interface |
| **warp** | Modern terminal lovers - prompt always visible |
| **compact** | Minimalists - maximum space for output |

**Recommendation:** Start with `classic` (it's the default), then try `warp` - you might like it!

---

## Other UI Settings

```json
{
  "ui": {
    "mode": "classic",
    "theme": "monokai",              // Color scheme
    "show_git_status": true,         // Show git in prompt
    "syntax_highlight": true,        // Command highlighting
    "completion_enabled": true       // Auto-completion
  }
}
```

More options - see `.goshrc.example`
