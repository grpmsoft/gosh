package repl

import (
	"bytes"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// ShellInput - кастомный input с поддержкой синтаксической подсветки
type ShellInput struct {
	value       string // Текущее значение
	cursorPos   int    // Позиция курсора (в рунах)
	width       int    // Ширина поля
	placeholder string // Placeholder текст

	// Syntax highlighting
	lexer     chroma.Lexer
	formatter chroma.Formatter
	style     *chroma.Style
}

// NewShellInput создает новый input с поддержкой подсветки
func NewShellInput() ShellInput {
	lexer := lexers.Get("bash")
	if lexer == nil {
		lexer = lexers.Fallback
	}

	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	style := styles.Get("monokai")
	if style == nil {
		style = styles.Fallback
	}

	return ShellInput{
		value:       "",
		cursorPos:   0,
		width:       80,
		placeholder: "",
		lexer:       lexer,
		formatter:   formatter,
		style:       style,
	}
}

// SetValue устанавливает значение
func (si *ShellInput) SetValue(value string) {
	si.value = value
	runes := []rune(value)
	if si.cursorPos > len(runes) {
		si.cursorPos = len(runes)
	}
}

// Value возвращает текущее значение
func (si *ShellInput) Value() string {
	return si.value
}

// SetWidth устанавливает ширину
func (si *ShellInput) SetWidth(width int) {
	si.width = width
}

// SetPlaceholder устанавливает placeholder
func (si *ShellInput) SetPlaceholder(placeholder string) {
	si.placeholder = placeholder
}

// Reset сбрасывает input
func (si *ShellInput) Reset() {
	si.value = ""
	si.cursorPos = 0
}

// CursorStart перемещает курсор в начало
func (si *ShellInput) CursorStart() {
	si.cursorPos = 0
}

// CursorEnd перемещает курсор в конец
func (si *ShellInput) CursorEnd() {
	si.cursorPos = len([]rune(si.value))
}

// applySyntaxHighlight применяет подсветку синтаксиса
func (si *ShellInput) applySyntaxHighlight(text string) string {
	if text == "" {
		return ""
	}

	iterator, err := si.lexer.Tokenise(nil, text)
	if err != nil {
		return text
	}

	var buf bytes.Buffer
	err = si.formatter.Format(&buf, si.style, iterator)
	if err != nil {
		return text
	}

	result := strings.TrimRight(buf.String(), "\n")
	return result
}

// View рендерит input с подсветкой и курсором
func (si ShellInput) View() string {
	if si.value == "" && si.placeholder != "" {
		// Показываем placeholder серым цветом
		return "\033[90m" + si.placeholder + "\033[0m"
	}

	runes := []rune(si.value)

	// Разделяем на часть до курсора и после
	beforeCursor := string(runes[:si.cursorPos])
	afterCursor := ""
	cursorChar := " " // Пустой курсор по умолчанию

	if si.cursorPos < len(runes) {
		cursorChar = string(runes[si.cursorPos])
		if si.cursorPos+1 < len(runes) {
			afterCursor = string(runes[si.cursorPos+1:])
		}
	}

	// Применяем подсветку к частям текста для правильной вставки курсора
	highlightedBefore := si.applySyntaxHighlight(beforeCursor)
	highlightedAfter := si.applySyntaxHighlight(afterCursor)

	// Курсор - инверсный символ
	cursor := "\033[7m" + cursorChar + "\033[0m"

	return highlightedBefore + cursor + highlightedAfter
}

// Update обрабатывает сообщения
func (si ShellInput) Update(msg tea.Msg) (ShellInput, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("left", "ctrl+b"))):
			if si.cursorPos > 0 {
				si.cursorPos--
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("right", "ctrl+f"))):
			runes := []rune(si.value)
			if si.cursorPos < len(runes) {
				si.cursorPos++
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("home", "ctrl+a"))):
			si.CursorStart()

		case key.Matches(msg, key.NewBinding(key.WithKeys("end", "ctrl+e"))):
			si.CursorEnd()

		case key.Matches(msg, key.NewBinding(key.WithKeys("backspace"))):
			if si.cursorPos > 0 {
				runes := []rune(si.value)
				si.value = string(runes[:si.cursorPos-1]) + string(runes[si.cursorPos:])
				si.cursorPos--
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("delete", "ctrl+d"))):
			runes := []rune(si.value)
			if si.cursorPos < len(runes) {
				si.value = string(runes[:si.cursorPos]) + string(runes[si.cursorPos+1:])
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+u"))):
			// Удалить всё до курсора
			runes := []rune(si.value)
			si.value = string(runes[si.cursorPos:])
			si.cursorPos = 0

		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+k"))):
			// Удалить всё после курсора
			runes := []rune(si.value)
			si.value = string(runes[:si.cursorPos])

		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+w"))):
			// Удалить слово перед курсором
			if si.cursorPos > 0 {
				runes := []rune(si.value)
				before := string(runes[:si.cursorPos])
				after := string(runes[si.cursorPos:])

				// Найти начало предыдущего слова
				trimmed := strings.TrimRight(before, " \t")
				lastSpace := strings.LastIndexAny(trimmed, " \t")

				if lastSpace == -1 {
					si.value = after
					si.cursorPos = 0
				} else {
					si.value = trimmed[:lastSpace+1] + after
					si.cursorPos = len([]rune(trimmed[:lastSpace+1]))
				}
			}

		default:
			// Обычный символ - вставляем в позицию курсора
			if msg.Type == tea.KeyRunes {
				runes := []rune(si.value)
				char := msg.Runes[0]

				si.value = string(runes[:si.cursorPos]) + string(char) + string(runes[si.cursorPos:])
				si.cursorPos++
			}
		}
	}

	return si, nil
}

// Blur - пока заглушка для совместимости
func (si *ShellInput) Blur() {
	// Не нужно для shell input
}

// Focus - пока заглушка для совместимости
func (si *ShellInput) Focus() tea.Cmd {
	return nil
}
