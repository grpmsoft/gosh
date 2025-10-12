package repl

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/grpmsoft/gosh/internal/application/execute"
	apphistory "github.com/grpmsoft/gosh/internal/application/history"
	appsession "github.com/grpmsoft/gosh/internal/application/session"
	"github.com/grpmsoft/gosh/internal/domain/command"
	"github.com/grpmsoft/gosh/internal/domain/config"
	"github.com/grpmsoft/gosh/internal/domain/history"
	"github.com/grpmsoft/gosh/internal/domain/process"
	"github.com/grpmsoft/gosh/internal/domain/session"
	"github.com/grpmsoft/gosh/internal/infrastructure/executor"
	historyInfra "github.com/grpmsoft/gosh/internal/infrastructure/history"
	"github.com/grpmsoft/gosh/internal/interfaces/parser"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

// Model представляет состояние REPL (Elm Architecture)
type Model struct {
	// Core components
	textarea         textarea.Model
	viewport         viewport.Model
	sessionManager   *appsession.SessionManager
	executeUseCase   *execute.ExecuteCommandUseCase
	pipelineExecutor *executor.OSPipelineExecutor
	commandExecutor  *executor.OSCommandExecutor
	currentSession   *session.Session
	logger           *slog.Logger
	ctx              context.Context
	config           *config.Config // Конфигурация

	// State
	output           []string                            // Вывод команд (прокручивается вверх как в терминале)
	historyNavigator *history.Navigator                  // Navigator for Up/Down arrow keys
	historyRepo      *historyInfra.FileHistoryRepository // For persistence
	addToHistoryUC   *apphistory.AddToHistoryUseCase     // Auto-save use case
	maxOutputLines   int
	width            int
	height           int
	ready            bool
	quitting         bool
	executing        bool
	lastExitCode     int
	startTime        time.Time
	gitBranch        string
	gitDirty         bool

	// Spinner для выполнения
	executingSpinner spinner.Model

	// Tab completion
	completions      []string
	completionIndex  int
	completionActive bool
	beforeCompletion string // Текст до нажатия Tab

	// Input state (для кастомного рендеринга)
	inputText string
	cursorPos int

	// Scrolling
	autoScroll bool // Автопрокрутка вниз при новых сообщениях

	// Help overlay
	showingHelp bool // Флаг отображения help overlay

	// Styles
	styles Styles
}

// Styles содержит все стили для UI (PowerShell/Git Bash inspired)
type Styles struct {
	// Prompt styles
	PromptUser     lipgloss.Style
	PromptPath     lipgloss.Style
	PromptGit      lipgloss.Style
	PromptGitDirty lipgloss.Style
	PromptArrow    lipgloss.Style
	PromptError    lipgloss.Style

	// Output styles
	Output    lipgloss.Style
	OutputErr lipgloss.Style

	// Executing spinner
	Executing lipgloss.Style

	// Completion hint
	CompletionHint lipgloss.Style

	// Syntax highlighting (inline в textarea)
	SyntaxCommand lipgloss.Style // Первое слово - команда
	SyntaxOption  lipgloss.Style // --option или -o
	SyntaxArg     lipgloss.Style // Обычные аргументы
	SyntaxString  lipgloss.Style // "строки в кавычках"
}

// commandExecutedMsg сообщение о выполненной команде
type commandExecutedMsg struct {
	output   string
	err      error
	exitCode int
}

// NewBubbleteaREPL создает новый bubbletea REPL
func NewBubbleteaREPL(
	sessionManager *appsession.SessionManager,
	executeUseCase *execute.ExecuteCommandUseCase,
	logger *slog.Logger,
	ctx context.Context,
	cfg *config.Config,
) (*Model, error) {
	// Создаем сессию
	sess, err := sessionManager.CreateSession(uuid.New().String())
	if err != nil {
		return nil, err
	}

	// Создаем textarea (для multiline с Alt+Enter)
	ta := textarea.New()
	ta.Placeholder = ""
	ta.Focus()
	ta.CharLimit = 0
	ta.SetWidth(80)
	ta.SetHeight(1) // Начинаем с одной строки
	ta.Prompt = ""  // Мы сами отрисуем prompt
	ta.ShowLineNumbers = false

	// Стили для textarea
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.Base = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	// Создаем стили
	styles := makeProfessionalStyles()

	// Создаем spinner
	execSpinner := spinner.New()
	execSpinner.Spinner = spinner.Dot
	execSpinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("12")) // Blue

	// Создаем viewport для прокрутки
	vp := viewport.New(80, 24)
	vp.MouseWheelEnabled = true
	// Отключаем дефолтные клавиши up/down (нужны для истории команд)
	vp.KeyMap.Up.SetEnabled(false)
	vp.KeyMap.Down.SetEnabled(false)
	// PageUp/PageDown оставляем включёнными

	// Create history repository and use cases
	historyFilePath := getHistoryFilePath()
	historyRepo := historyInfra.NewFileHistoryRepository(historyFilePath)

	// Load history from file into session
	loadHistoryUC := apphistory.NewLoadHistoryUseCase(historyRepo)
	if err := loadHistoryUC.Execute(sess.History()); err != nil {
		logger.Warn("Failed to load history", "error", err)
	}

	// Load .goshrc (aliases and environment)
	goshrcPath, err := config.GetDefaultGoshrcPath()
	if err != nil {
		logger.Warn("Failed to get .goshrc path", "error", err)
	} else {
		goshrcService := config.NewGoshrcService(goshrcPath)
		goshrcData, err := goshrcService.Load()
		if err != nil {
			logger.Warn("Failed to load .goshrc", "error", err)
		} else if goshrcData != nil {
			// Загружаем алиасы в сессию
			for name, command := range goshrcData.Aliases {
				_ = sess.SetAlias(name, command)
			}
			logger.Info("Loaded .goshrc", "aliases", len(goshrcData.Aliases), "env_vars", len(goshrcData.Environment))
		}
	}

	// Create navigator and use case for adding to history
	historyNavigator := sess.NewHistoryNavigator()
	addToHistoryUC := apphistory.NewAddToHistoryUseCase(sess.History(), historyRepo)

	// Create executors for handling commands
	pipelineExecutor := executor.NewOSPipelineExecutor(logger)
	commandExecutor := executor.NewOSCommandExecutor(logger)

	m := &Model{
		textarea:         ta,
		viewport:         vp,
		sessionManager:   sessionManager,
		executeUseCase:   executeUseCase,
		pipelineExecutor: pipelineExecutor,
		commandExecutor:  commandExecutor,
		currentSession:   sess,
		logger:           logger,
		ctx:              ctx,
		config:           cfg,
		output:           make([]string, 0),
		historyNavigator: historyNavigator,
		historyRepo:      historyRepo,
		addToHistoryUC:   addToHistoryUC,
		maxOutputLines:   10000,
		ready:            false,
		quitting:         false,
		executing:        false,
		startTime:        time.Now(),
		styles:           styles,
		executingSpinner: execSpinner,
		completions:      []string{},
		completionIndex:  -1,
		completionActive: false,
		beforeCompletion: "",
		inputText:        "",
		cursorPos:        0,
		autoScroll:       true, // По умолчанию авто-скролл вниз
		showingHelp:      false,
	}

	// Определяем Git статус
	m.updateGitInfo()

	// Welcome message (цветное через ANSI)
	m.addOutputRaw("\033[1;33mGoSh\033[0m - Go Shell \033[90m(Git Bash inspired)\033[0m")
	m.addOutputRaw("Press \033[1;36mF1\033[0m or \033[1;36m?\033[0m for help, \033[1;31m'exit'\033[0m to quit")
	m.addOutputRaw("\033[90mSyntax: \033[1;33mcommands\033[0m yellow, \033[90moptions\033[0m gray, \033[32marguments\033[0m green\033[0m")
	m.addOutputRaw("\033[90mScroll: PgUp/PgDn or Mouse Wheel\033[0m")

	// Подсказка про переключение режимов
	if cfg.UI.AllowModeSwitching {
		m.addOutputRaw("\033[90mUI Modes: Ctrl+F5=Classic, Ctrl+F6=Warp, Ctrl+F7=Compact, Ctrl+F8=Chat\033[0m")
	}
	m.addOutputRaw("")

	// Инициализируем viewport content
	m.updateViewportContent()

	return m, nil
}

// Init инициализирует модель (Elm Architecture)
func (m Model) Init() tea.Cmd {
	return textarea.Blink
}

// Update обрабатывает сообщения (Elm Architecture)
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		taCmd tea.Cmd
		vpCmd tea.Cmd
		spCmd tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.MouseMsg:
		// Обрабатываем mouse wheel для viewport
		if msg.Type == tea.MouseWheelUp || msg.Type == tea.MouseWheelDown {
			m.autoScroll = false // Отключаем автоскролл при ручной прокрутке
			m.viewport, vpCmd = m.viewport.Update(msg)
			return m, vpCmd
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textarea.SetWidth(msg.Width)

		// Обновляем размер viewport
		// Classic mode: промпт ВНУТРИ viewport → используем ВСЮ высоту
		// Другие режимы: промпт СНАРУЖИ → резервируем место снизу
		var viewportHeight int
		if m.config.UI.Mode == config.UIModeClassic {
			// Classic mode - промпт внутри, занимаем весь экран
			viewportHeight = msg.Height
		} else {
			// Warp/Compact/Chat - промпт снаружи, резервируем 2-3 строки
			viewportHeight = msg.Height - 3
		}

		if viewportHeight < 1 {
			viewportHeight = 1
		}
		m.viewport.Width = msg.Width
		m.viewport.Height = viewportHeight
		m.updateViewportContent()

		m.ready = true
		return m, nil

	case commandExecutedMsg:
		m.executing = false
		m.lastExitCode = msg.exitCode

		// Показываем output (stdout + stderr) если есть
		if msg.output != "" {
			// Разбиваем вывод на строки и выводим как есть (без стилей)
			lines := strings.Split(strings.TrimRight(msg.output, "\n"), "\n")
			for _, line := range lines {
				m.addOutputRaw(line)
			}
		}

		// Показываем дополнительную ошибку если есть (например "exit status 1")
		// Обычно msg.err содержит только exit status, а реальный stderr уже в msg.output
		if msg.err != nil && msg.output == "" {
			// Показываем ошибку только если нет output (чтобы не дублировать)
			m.addOutputRaw("\033[31mError: " + msg.err.Error() + "\033[0m")
		}

		// Обновляем Git статус после каждой команды
		m.updateGitInfo()

		// Обновляем viewport content и скроллим вниз если autoScroll
		m.updateViewportContent()
		if m.autoScroll {
			m.viewport.GotoBottom()
		}

		return m, nil
	}

	// Обновляем textarea
	m.textarea, taCmd = m.textarea.Update(msg)

	// Синхронизируем наш input state с textarea
	m.inputText = m.textarea.Value()
	// Курсор всегда в конце после обычного ввода (textarea не дает API для позиции)
	m.cursorPos = len([]rune(m.inputText))

	// Update viewport (для прокрутки PageUp/PageDown)
	m.viewport, vpCmd = m.viewport.Update(msg)

	// Update spinner if executing
	if m.executing {
		m.executingSpinner, spCmd = m.executingSpinner.Update(msg)
	}

	return m, tea.Batch(taCmd, vpCmd, spCmd)
}

// handleKeyPress обрабатывает нажатия клавиш
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// ESC - закрыть help overlay (если открыт)
	if msg.String() == "esc" && m.showingHelp {
		m.showingHelp = false
		return m, nil
	}

	// F1 или ? - открыть help overlay
	if msg.String() == "f1" || msg.String() == "?" {
		m.showingHelp = true
		return m, nil
	}

	// Если показываем help - блокируем остальные клавиши
	if m.showingHelp {
		return m, nil
	}

	switch msg.String() {
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	case "ctrl+d":
		if m.textarea.Value() == "" {
			m.quitting = true
			return m, tea.Quit
		}

	case "enter":
		// Обычный Enter - выполнить команду
		m.autoScroll = true // Включаем автоскролл при выполнении команды
		return m.executeCommand()

	case "alt+enter":
		// Alt+Enter - добавить новую строку (multiline)
		currentHeight := m.textarea.Height()
		if currentHeight < 10 {
			m.textarea.SetHeight(currentHeight + 1)
		}
		// Позволяем textarea обработать вставку новой строки
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd

	case "up", "down":
		// История команд
		return m.navigateHistory(msg.String())

	case "tab":
		// Tab-completion
		return m.handleTabCompletion()

	case "ctrl+l":
		// Очистить экран
		m.output = make([]string, 0)
		m.updateViewportContent()
		m.autoScroll = true
		return m, tea.ClearScreen

	case "pgup", "pgdown", "f", "b":
		// Прокрутка viewport
		m.autoScroll = false
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd

	// Горячие клавиши для переключения UI режимов (Ctrl+F5-F8)
	case "ctrl+f5", "ctrl+f6", "ctrl+f7", "ctrl+f8":
		if m.config.UI.AllowModeSwitching {
			return m.switchUIMode(msg.String())
		}
	}

	// Сброс completion при любом другом вводе
	if m.completionActive {
		m.completionActive = false
		m.completions = []string{}
		m.completionIndex = -1
		m.beforeCompletion = ""
	}

	// При любом вводе возвращаем автоскролл
	if msg.Type == tea.KeyRunes {
		m.autoScroll = true
	}

	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

// switchUIMode переключает режим UI
func (m Model) switchUIMode(key string) (tea.Model, tea.Cmd) {
	var newMode config.UIMode

	switch key {
	case "ctrl+f5":
		newMode = config.UIModeClassic
	case "ctrl+f6":
		newMode = config.UIModeWarp
	case "ctrl+f7":
		newMode = config.UIModeCompact
	case "ctrl+f8":
		newMode = config.UIModeChat
	default:
		return m, nil
	}

	// Если уже в этом режиме - ничего не делаем
	if m.config.UI.Mode == newMode {
		return m, nil
	}

	// Меняем режим
	oldMode := m.config.UI.Mode
	m.config.UI.Mode = newMode

	// ВАЖНО: пересчитываем размер viewport в зависимости от нового режима
	// Classic mode: промпт ВНУТРИ viewport → используем ВСЮ высоту
	// Другие режимы: промпт СНАРУЖИ → резервируем 2-3 строки
	var viewportHeight int
	if newMode == config.UIModeClassic {
		// Classic mode - промпт внутри, занимаем весь экран
		viewportHeight = m.height
	} else {
		// Warp/Compact/Chat - промпт снаружи, резервируем 2-3 строки
		viewportHeight = m.height - 3
	}

	if viewportHeight < 1 {
		viewportHeight = 1
	}
	m.viewport.Height = viewportHeight

	// Логируем переключение
	m.logger.Info("UI mode switched", "from", oldMode, "to", newMode)

	// Добавляем уведомление в output
	m.addOutputRaw(fmt.Sprintf("\033[90m[UI Mode: %s]\033[0m", newMode))
	m.updateViewportContent()

	// Скроллим вниз если включен автоскролл
	if m.autoScroll {
		m.viewport.GotoBottom()
	}

	return m, nil
}

// handleModeCommand обрабатывает команду :mode для переключения UI режимов
func (m Model) handleModeCommand(commandLine string) (tea.Model, tea.Cmd) {
	// Проверяем включено ли переключение режимов
	if !m.config.UI.AllowModeSwitching {
		m.addOutputRaw("\033[31mError: UI mode switching is disabled in config\033[0m")
		m.updateViewportContent()
		if m.autoScroll {
			m.viewport.GotoBottom()
		}
		return m, nil
	}

	// Парсим аргументы команды
	parts := strings.Fields(commandLine)

	// Если только ":mode" без аргументов - показываем текущий режим
	if len(parts) == 1 {
		m.addOutputRaw(fmt.Sprintf("\033[90mCurrent UI mode: \033[1;32m%s\033[0m", m.config.UI.Mode))
		m.addOutputRaw("\033[90mAvailable modes: classic, warp, compact, chat\033[0m")
		m.addOutputRaw("\033[90mUsage: :mode <name>\033[0m")
		m.updateViewportContent()
		if m.autoScroll {
			m.viewport.GotoBottom()
		}
		return m, nil
	}

	// Получаем имя режима
	modeName := strings.ToLower(parts[1])

	// Маппинг имён на режимы
	var newMode config.UIMode
	switch modeName {
	case "classic":
		newMode = config.UIModeClassic
	case "warp":
		newMode = config.UIModeWarp
	case "compact":
		newMode = config.UIModeCompact
	case "chat":
		newMode = config.UIModeChat
	default:
		m.addOutputRaw(fmt.Sprintf("\033[31mError: unknown mode '%s'\033[0m", modeName))
		m.addOutputRaw("\033[90mAvailable modes: classic, warp, compact, chat\033[0m")
		m.updateViewportContent()
		if m.autoScroll {
			m.viewport.GotoBottom()
		}
		return m, nil
	}

	// Если уже в этом режиме - просто уведомляем
	if m.config.UI.Mode == newMode {
		m.addOutputRaw(fmt.Sprintf("\033[90mAlready in %s mode\033[0m", newMode))
		m.updateViewportContent()
		if m.autoScroll {
			m.viewport.GotoBottom()
		}
		return m, nil
	}

	// Меняем режим
	oldMode := m.config.UI.Mode
	m.config.UI.Mode = newMode

	// ВАЖНО: пересчитываем размер viewport в зависимости от нового режима
	var viewportHeight int
	if newMode == config.UIModeClassic {
		viewportHeight = m.height
	} else {
		viewportHeight = m.height - 3
	}

	if viewportHeight < 1 {
		viewportHeight = 1
	}
	m.viewport.Height = viewportHeight

	// Логируем переключение
	m.logger.Info("UI mode switched via :mode command", "from", oldMode, "to", newMode)

	// Добавляем уведомление в output
	m.addOutputRaw(fmt.Sprintf("\033[90m[UI Mode: %s]\033[0m", newMode))
	m.updateViewportContent()

	// Скроллим вниз если включен автоскролл
	if m.autoScroll {
		m.viewport.GotoBottom()
	}

	return m, nil
}

// handleTabCompletion обрабатывает Tab-completion
func (m Model) handleTabCompletion() (tea.Model, tea.Cmd) {
	input := m.textarea.Value()

	// Первое нажатие Tab - генерируем completions
	if !m.completionActive {
		m.beforeCompletion = input
		m.completions = m.generateCompletions(input)
		m.completionIndex = -1

		if len(m.completions) == 0 {
			return m, nil
		}

		m.completionActive = true
		m.completionIndex = 0
		m.textarea.SetValue(m.completions[0])
		// Синхронизируем input state
		m.inputText = m.completions[0]
		m.cursorPos = len([]rune(m.inputText))
		return m, nil
	}

	// Повторные нажатия Tab - переключаем между вариантами
	if len(m.completions) > 0 {
		m.completionIndex = (m.completionIndex + 1) % len(m.completions)
		m.textarea.SetValue(m.completions[m.completionIndex])
		// Синхронизируем input state
		m.inputText = m.completions[m.completionIndex]
		m.cursorPos = len([]rune(m.inputText))
	}

	return m, nil
}

// generateCompletions генерирует варианты автодополнения
func (m *Model) generateCompletions(input string) []string {
	completions := []string{}

	if input == "" {
		return completions
	}

	// Парсим ввод для определения что дополнять
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return completions
	}

	// Первое слово - команда
	if len(parts) == 1 {
		prefix := parts[0]

		// Встроенные команды
		builtins := []string{"cd", "pwd", "echo", "exit", "help", "clear", "export", "unset", "env", "type", "alias", "unalias"}
		for _, cmd := range builtins {
			if strings.HasPrefix(cmd, prefix) {
				completions = append(completions, cmd)
			}
		}

		// Алиасы
		aliases := m.currentSession.GetAllAliases()
		for aliasName := range aliases {
			if strings.HasPrefix(aliasName, prefix) {
				completions = append(completions, aliasName)
			}
		}

		// PATH команды можно добавить позже
		return completions
	}

	// Остальные слова - файлы/директории
	lastPart := parts[len(parts)-1]
	dirPath := filepath.Dir(lastPart)
	baseName := filepath.Base(lastPart)

	if dirPath == "." {
		dirPath = m.currentSession.WorkingDirectory()
	} else if !filepath.IsAbs(dirPath) {
		dirPath = filepath.Join(m.currentSession.WorkingDirectory(), dirPath)
	}

	// Читаем директорию
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return completions
	}

	// Фильтруем по prefix
	prefix := input[:len(input)-len(baseName)]
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, baseName) {
			completion := prefix + name
			if entry.IsDir() {
				completion += string(filepath.Separator)
			}
			completions = append(completions, completion)
		}
	}

	return completions
}

// expandAliases рекурсивно раскрывает алиасы в команде
// Возвращает раскрытую команду или ошибку при циклической зависимости
func (m *Model) expandAliases(commandLine string, depth int) (string, error) {
	const maxDepth = 10 // Защита от бесконечной рекурсии

	// Проверка глубины рекурсии
	if depth > maxDepth {
		return "", fmt.Errorf("alias expansion exceeded maximum depth (possible recursive alias)")
	}

	// Извлекаем первое слово (имя команды)
	parts := strings.Fields(commandLine)
	if len(parts) == 0 {
		return commandLine, nil
	}

	cmdName := parts[0]

	// Проверяем является ли первое слово алиасом
	aliasCommand, isAlias := m.currentSession.GetAlias(cmdName)
	if !isAlias {
		// Не алиас - возвращаем как есть
		return commandLine, nil
	}

	// Раскрываем алиас
	// Если у команды были аргументы, добавляем их к раскрытому алиасу
	var expandedCommand string
	if len(parts) > 1 {
		// Алиас + оригинальные аргументы
		args := strings.Join(parts[1:], " ")
		expandedCommand = aliasCommand + " " + args
	} else {
		// Только алиас без аргументов
		expandedCommand = aliasCommand
	}

	// Рекурсивно раскрываем алиасы в результате (на случай alias на alias)
	return m.expandAliases(expandedCommand, depth+1)
}

// executeCommand выполняет введенную команду
func (m Model) executeCommand() (tea.Model, tea.Cmd) {
	value := strings.TrimSpace(m.textarea.Value())

	// Пустая команда
	if value == "" {
		return m, nil
	}

	// Сброс completion
	m.completionActive = false
	m.completions = []string{}
	m.completionIndex = -1
	m.beforeCompletion = ""

	// Добавляем в историю через use case (auto-save если настроено)
	if err := m.addToHistoryUC.Execute(value); err != nil {
		m.logger.Warn("Failed to add command to history", "error", err)
	}

	// Создаем новый navigator для сброса позиции
	m.historyNavigator = m.currentSession.NewHistoryNavigator()

	// Показываем команду в output с prompt и syntax highlighting (только ANSI коды)
	m.addOutputRaw(m.renderPromptForHistoryANSI() + m.applySyntaxHighlight(value))

	// Очищаем textarea и возвращаем высоту к 1
	m.textarea.SetValue("")
	m.textarea.SetHeight(1)

	// Синхронизируем input state
	m.inputText = ""
	m.cursorPos = 0

	// Встроенная команда exit
	if value == "exit" || value == "quit" {
		m.quitting = true
		return m, tea.Quit
	}

	// Встроенная команда clear
	if value == "clear" || value == "cls" {
		m.output = make([]string, 0)
		return m, tea.ClearScreen
	}

	// Встроенная команда help
	if value == "help" {
		m.showHelp()
		return m, nil
	}

	// Встроенная команда :mode для переключения UI режимов
	if strings.HasPrefix(value, ":mode ") || value == ":mode" {
		return m.handleModeCommand(value)
	}

	// Раскрываем алиасы (если команда является алиасом)
	expandedValue, err := m.expandAliases(value, 0)
	if err != nil {
		m.addOutputRaw("\033[31mError: " + err.Error() + "\033[0m")
		m.updateViewportContent()
		if m.autoScroll {
			m.viewport.GotoBottom()
		}
		return m, nil
	}

	// Используем раскрытую команду для дальнейшего выполнения
	value = expandedValue

	// Определяем тип команды и способ выполнения
	cmdName, cmdArgs := m.extractCommandName(value)

	// Проверяем является ли это shell скриптом
	scriptPath, isScript := m.isShellScript(cmdName)

	if isScript {
		// Shell скрипт (.sh/.bash)
		if m.isInteractiveCommand(cmdName) {
			// Интерактивный скрипт (с read, clear, menu) - через bash + tea.ExecProcess
			return m, m.execInteractiveCommand(value)
		} else {
			// Обычный скрипт - выполняем НАТИВНО через mvdan.cc/sh
			m.executing = true
			return m, m.executeShellScriptNative(scriptPath, cmdArgs)
		}
	}

	// Интерактивная команда (vim, ssh, etc.) - через tea.ExecProcess
	if m.isInteractiveCommand(cmdName) {
		return m, m.execInteractiveCommand(value)
	}

	// Обычная команда - выполняем асинхронно с захватом вывода
	m.executing = true
	return m, m.execCommandAsync(value)
}

// showHelp показывает справку (текстовая версия для команды help)
func (m *Model) showHelp() {
	m.addOutputRaw("\033[1;33mGoSh - Built-in Commands:\033[0m")
	m.addOutputRaw("  cd <dir>     - Change directory")
	m.addOutputRaw("  pwd          - Print working directory")
	m.addOutputRaw("  echo <text>  - Print text")
	m.addOutputRaw("  export VAR=value - Set environment variable")
	m.addOutputRaw("  unset VAR    - Unset environment variable")
	m.addOutputRaw("  env          - Show environment")
	m.addOutputRaw("  type <cmd>   - Show command type")
	m.addOutputRaw("  alias        - List/create aliases")
	m.addOutputRaw("  unalias      - Remove alias")
	m.addOutputRaw("  jobs         - List background jobs")
	m.addOutputRaw("  fg, bg       - Foreground/background job control")
	m.addOutputRaw("  clear, cls   - Clear screen")
	m.addOutputRaw("  help         - Show this help")
	m.addOutputRaw("  exit, quit   - Exit shell")
	m.addOutputRaw("")

	// UI режимы (если разрешено переключение)
	if m.config.UI.AllowModeSwitching {
		m.addOutputRaw("\033[1;33mUI Mode Switching:\033[0m")
		m.addOutputRaw("  :mode        - Show current UI mode")
		m.addOutputRaw("  :mode <name> - Switch UI mode (classic/warp/compact/chat)")
		m.addOutputRaw("")
	}

	m.addOutputRaw("\033[1;36mPress F1 or ? for visual keyboard shortcuts\033[0m")
	m.updateViewportContent()
}

// execCommandAsync выполняет команду в фоне
func (m *Model) execCommandAsync(commandLine string) tea.Cmd {
	return func() tea.Msg {
		// Парсим команду
		cmd, pipe, err := parser.ParseCommandLine(commandLine)
		if err != nil {
			return commandExecutedMsg{
				err:      err,
				exitCode: 1,
			}
		}

		// Single command execution through OSCommandExecutor (handles redirections)
		if cmd != nil {
			// Подготавливаем команду (проверяем скрипты и добавляем интерпретатор если нужно)
			cmdName, cmdArgs := m.prepareCommand(cmd.Name(), cmd.Args())

			// Обновляем команду с подготовленными именем и аргументами
			preparedCmd, err := command.NewCommand(cmdName, cmdArgs, command.TypeExternal)
			if err != nil {
				return commandExecutedMsg{
					err:      err,
					exitCode: 1,
				}
			}

			// Копируем перенаправления из оригинальной команды
			for _, redir := range cmd.Redirections() {
				if err := preparedCmd.AddRedirection(redir); err != nil {
					return commandExecutedMsg{
						err:      err,
						exitCode: 1,
					}
				}
			}

			// Выполняем команду через OSCommandExecutor (поддерживает redirections)
			proc, err := m.commandExecutor.Execute(m.ctx, preparedCmd, m.currentSession)
			if err != nil {
				return commandExecutedMsg{
					err:      err,
					exitCode: 1,
				}
			}

			// Комбинируем stdout и stderr
			output := proc.Stdout()
			if stderr := proc.Stderr(); stderr != "" {
				if output != "" {
					output += "\n"
				}
				output += stderr
			}

			// Определяем exitCode и ошибку
			exitCode := int(proc.ExitCode())
			var execErr error
			if proc.State() == process.StateFailed {
				execErr = proc.Error()
			}

			return commandExecutedMsg{
				output:   output,
				err:      execErr,
				exitCode: exitCode,
			}
		}

		if pipe != nil {
			// Выполняем pipeline через OSPipelineExecutor
			processes, err := m.pipelineExecutor.Execute(m.ctx, pipe.Commands(), m.currentSession)
			if err != nil {
				return commandExecutedMsg{
					err:      fmt.Errorf("pipeline execution failed: %w", err),
					exitCode: 1,
				}
			}

			// Получаем последний процесс (он содержит финальный вывод)
			if len(processes) == 0 {
				return commandExecutedMsg{
					output:   "",
					exitCode: 0,
				}
			}

			lastProcess := processes[len(processes)-1]

			// Комбинируем stdout и stderr
			output := lastProcess.Stdout()
			if stderr := lastProcess.Stderr(); stderr != "" {
				if output != "" {
					output += "\n"
				}
				output += stderr
			}

			// Проверяем статус последнего процесса
			exitCode := int(lastProcess.ExitCode())
			var execErr error
			if lastProcess.State() == process.StateFailed {
				execErr = lastProcess.Error()
			}

			return commandExecutedMsg{
				output:   output,
				err:      execErr,
				exitCode: exitCode,
			}
		}

		return commandExecutedMsg{
			output:   "",
			exitCode: 0,
		}
	}
}

// prepareCommand подготавливает команду для выполнения
// Определяет скрипты и добавляет нужный интерпретатор (sh, bash, cmd, powershell)
func (m *Model) prepareCommand(cmdName string, cmdArgs []string) (string, []string) {
	// Проверяем является ли команда файлом скрипта
	var scriptPath string

	// Если путь относительный или абсолютный (универсальная проверка для всех ОС)
	if strings.HasPrefix(cmdName, ".") || strings.ContainsRune(cmdName, filepath.Separator) || filepath.IsAbs(cmdName) {
		// Проверяем существование файла
		if filepath.IsAbs(cmdName) {
			scriptPath = cmdName
		} else {
			scriptPath = filepath.Join(m.currentSession.WorkingDirectory(), cmdName)
		}

		// Проверяем существует ли файл
		if _, err := os.Stat(scriptPath); err != nil {
			// Файл не существует - возвращаем как есть (будет ошибка exec)
			return cmdName, cmdArgs
		}

		// Определяем тип скрипта по расширению
		ext := strings.ToLower(filepath.Ext(scriptPath))

		switch ext {
		case ".sh", ".bash":
			// Shell script - запускаем через sh или bash
			// Проверяем доступность bash, иначе sh
			interpreter := "sh"
			if _, err := exec.LookPath("bash"); err == nil {
				interpreter = "bash"
			}
			// Git Bash на Windows понимает Windows пути напрямую!
			// Передаем путь как есть (не конвертируем)
			// Возвращаем: bash script.sh args...
			newArgs := append([]string{scriptPath}, cmdArgs...)
			return interpreter, newArgs

		case ".bat", ".cmd":
			// Windows batch - запускаем через cmd /c
			newArgs := append([]string{"/c", scriptPath}, cmdArgs...)
			return "cmd", newArgs

		case ".ps1":
			// PowerShell script - запускаем через powershell -File
			newArgs := append([]string{"-File", scriptPath}, cmdArgs...)
			return "powershell", newArgs
		}
	}

	// Не скрипт или не найден - возвращаем как есть
	return cmdName, cmdArgs
}

// isShellScript проверяет является ли команда .sh/.bash скриптом
// Возвращает (путь к скрипту, true) если это скрипт, иначе ("", false)
func (m *Model) isShellScript(cmdName string) (string, bool) {
	// Проверяем является ли это файлом (путь относительный или абсолютный)
	if !strings.HasPrefix(cmdName, ".") && !strings.ContainsRune(cmdName, filepath.Separator) && !filepath.IsAbs(cmdName) {
		// Не похоже на путь к файлу
		return "", false
	}

	// Получаем полный путь
	var scriptPath string
	if filepath.IsAbs(cmdName) {
		scriptPath = cmdName
	} else {
		scriptPath = filepath.Join(m.currentSession.WorkingDirectory(), cmdName)
	}

	// Проверяем существует ли файл
	if _, err := os.Stat(scriptPath); err != nil {
		return "", false
	}

	// Проверяем расширение
	ext := strings.ToLower(filepath.Ext(scriptPath))
	if ext == ".sh" || ext == ".bash" {
		return scriptPath, true
	}

	return "", false
}

// extractCommandName извлекает имя команды из строки
func (m *Model) extractCommandName(commandLine string) (string, []string) {
	// Парсим команду
	cmd, _, err := parser.ParseCommandLine(commandLine)
	if err != nil || cmd == nil {
		// Fallback: берем первое слово
		parts := strings.Fields(commandLine)
		if len(parts) == 0 {
			return "", nil
		}
		return parts[0], parts[1:]
	}

	return cmd.Name(), cmd.Args()
}

// isInteractiveCommand определяет требует ли команда интерактивного терминала
func (m *Model) isInteractiveCommand(cmdName string) bool {
	// Проверяем является ли это скриптом (универсальная проверка для всех ОС)
	if strings.HasPrefix(cmdName, ".") || strings.ContainsRune(cmdName, filepath.Separator) || filepath.IsAbs(cmdName) {
		var scriptPath string
		if filepath.IsAbs(cmdName) {
			scriptPath = cmdName
		} else {
			scriptPath = filepath.Join(m.currentSession.WorkingDirectory(), cmdName)
		}

		// Проверяем расширение файла
		ext := strings.ToLower(filepath.Ext(scriptPath))
		switch ext {
		case ".sh", ".bash", ".bat", ".cmd", ".ps1":
			// Скрипты могут требовать интерактив (read, input, etc.)
			return true
		}
	}

	// Список известных интерактивных команд
	interactiveCommands := map[string]bool{
		"vi":     true,
		"vim":    true,
		"nvim":   true,
		"nano":   true,
		"emacs":  true,
		"less":   true,
		"more":   true,
		"top":    true,
		"htop":   true,
		"ssh":    true,
		"telnet": true,
		"ftp":    true,
		"sftp":   true,
		"python": true, // Python REPL
		"node":   true, // Node.js REPL
		"irb":    true, // Ruby REPL
		"psql":   true, // PostgreSQL
		"mysql":  true, // MySQL
		"mongo":  true, // MongoDB
	}

	// Проверяем базовое имя команды (без пути)
	baseName := filepath.Base(cmdName)
	baseName = strings.TrimSuffix(baseName, filepath.Ext(baseName))

	return interactiveCommands[baseName]
}

// executeShellScriptNative выполняет .sh/.bash скрипт нативно через mvdan.cc/sh
func (m *Model) executeShellScriptNative(scriptPath string, args []string) tea.Cmd {
	return func() tea.Msg {
		// Открываем файл скрипта
		file, err := os.Open(scriptPath)
		if err != nil {
			return commandExecutedMsg{
				err:      fmt.Errorf("failed to open script: %w", err),
				exitCode: 1,
			}
		}
		defer func() { _ = file.Close() }()

		// Парсим скрипт
		parser := syntax.NewParser()
		prog, err := parser.Parse(file, scriptPath)
		if err != nil {
			return commandExecutedMsg{
				err:      fmt.Errorf("failed to parse script: %w", err),
				exitCode: 1,
			}
		}

		// Создаем буферы для захвата вывода
		var stdout, stderr bytes.Buffer

		// Создаем интерпретатор с нашими настройками
		runner, err := interp.New(
			interp.StdIO(nil, &stdout, &stderr),                     // Захватываем stdout/stderr
			interp.Dir(m.currentSession.WorkingDirectory()),         // Рабочая директория
			interp.Env(expandEnv(m.currentSession)),                 // Переменные окружения
			interp.Params(append([]string{scriptPath}, args...)...), // Аргументы скрипта ($0, $1, ...)
		)
		if err != nil {
			return commandExecutedMsg{
				err:      fmt.Errorf("failed to create interpreter: %w", err),
				exitCode: 1,
			}
		}

		// Выполняем скрипт
		err = runner.Run(m.ctx, prog)

		// Определяем exit code (v3.12.0+ API)
		exitCode := 0
		if err != nil {
			// Новый API: ExitStatus возвращает uint8
			var exitStatus interp.ExitStatus
			if errors.As(err, &exitStatus) {
				exitCode = int(exitStatus)
			} else if err != nil {
				// Другая ошибка (не exit status)
				exitCode = 1
			}
		}

		// Комбинируем stdout и stderr
		output := stdout.String()
		if stderr.Len() > 0 {
			if output != "" {
				output += "\n"
			}
			output += stderr.String()
		}

		return commandExecutedMsg{
			output:   output,
			err:      err,
			exitCode: exitCode,
		}
	}
}

// sessionEnviron адаптер для session.Environment → expand.Environ
type sessionEnviron struct {
	sess *session.Session
}

func (e *sessionEnviron) Get(name string) expand.Variable {
	value, exists := e.sess.Environment().Get(name)
	if !exists {
		// Переменная не найдена - возвращаем пустую
		return expand.Variable{}
	}
	return expand.Variable{
		Exported: true,
		Kind:     expand.String,
		Str:      value,
	}
}

func (e *sessionEnviron) Each(fn func(name string, vr expand.Variable) bool) {
	// Итерация по всем переменным окружения
	envSlice := e.sess.Environment().ToSlice()
	for _, envVar := range envSlice {
		// Парсим "KEY=VALUE"
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) == 2 {
			name := parts[0]
			value := parts[1]
			vr := expand.Variable{
				Exported: true,
				Kind:     expand.String,
				Str:      value,
			}
			if !fn(name, vr) {
				break
			}
		}
	}
}

// expandEnv создает адаптер для session.Environment
func expandEnv(sess *session.Session) expand.Environ {
	return &sessionEnviron{sess: sess}
}

// execInteractiveCommand выполняет интерактивную команду через tea.ExecProcess
// Используется для скриптов требующих полный TTY (clear, read, menu)
func (m *Model) execInteractiveCommand(commandLine string) tea.Cmd {
	// Парсим команду
	cmd, pipe, err := parser.ParseCommandLine(commandLine)
	if err != nil {
		return func() tea.Msg {
			return commandExecutedMsg{
				err:      err,
				exitCode: 1,
			}
		}
	}

	// Пока не поддерживаем пайпы в интерактивном режиме
	if pipe != nil {
		return func() tea.Msg {
			return commandExecutedMsg{
				output:   "[Interactive pipes not yet supported]",
				exitCode: 1,
			}
		}
	}

	if cmd == nil {
		return func() tea.Msg {
			return commandExecutedMsg{
				output:   "",
				exitCode: 0,
			}
		}
	}

	// Подготавливаем команду (скрипты через интерпретаторы)
	cmdName, cmdArgs := m.prepareCommand(cmd.Name(), cmd.Args())

	// Создаем exec.Cmd с правильными настройками
	osCmd := exec.Command(cmdName, cmdArgs...)
	osCmd.Dir = m.currentSession.WorkingDirectory()
	osCmd.Env = m.currentSession.Environment().ToSlice()

	// Создаем exec.Cmd для интерактивного выполнения
	return tea.ExecProcess(osCmd, func(err error) tea.Msg {
		exitCode := 0
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = 1
			}
		}

		// Возвращаем сообщение о завершении
		// Вывод уже был показан напрямую в терминал
		return commandExecutedMsg{
			output:   "", // Пустой - вывод был интерактивным
			err:      err,
			exitCode: exitCode,
		}
	})
}

// navigateHistory навигация по истории команд через History.Navigator
func (m Model) navigateHistory(direction string) (tea.Model, tea.Cmd) {
	var cmd string
	var ok bool

	if direction == "up" {
		cmd, ok = m.historyNavigator.Backward()
	} else if direction == "down" {
		cmd, ok = m.historyNavigator.Forward()
	}

	// Если навигация успешна, устанавливаем значение
	if ok || direction == "down" {
		m.textarea.SetValue(cmd)
		if cmd != "" {
			m.textarea.CursorEnd()
		}
		// Синхронизируем input state
		m.inputText = cmd
		m.cursorPos = len([]rune(m.inputText))
	}

	return m, nil
}

// addOutput добавляет строку в output (с применением стиля)
func (m *Model) addOutput(line string) {
	m.output = append(m.output, line)

	// Ограничиваем размер output
	if len(m.output) > m.maxOutputLines {
		m.output = m.output[len(m.output)-m.maxOutputLines:]
	}
}

// addOutputRaw добавляет строку в output БЕЗ применения стилей (для ANSI кодов)
func (m *Model) addOutputRaw(line string) {
	m.output = append(m.output, line)

	// Ограничиваем размер output
	if len(m.output) > m.maxOutputLines {
		m.output = m.output[len(m.output)-m.maxOutputLines:]
	}
}

// updateViewportContent обновляет содержимое viewport из output
func (m *Model) updateViewportContent() {
	content := strings.Join(m.output, "\n")
	m.viewport.SetContent(content)
}

// updateGitInfo обновляет информацию о Git репозитории
func (m *Model) updateGitInfo() {
	m.gitBranch = ""
	m.gitDirty = false

	workDir := m.currentSession.WorkingDirectory()

	// Проверяем наличие .git
	gitDir := filepath.Join(workDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return
	}

	// Получаем текущую ветку
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = workDir
	if output, err := cmd.Output(); err == nil {
		m.gitBranch = strings.TrimSpace(string(output))
	}

	// Проверяем есть ли изменения
	cmd = exec.Command("git", "status", "--porcelain")
	cmd.Dir = workDir
	if output, err := cmd.Output(); err == nil {
		m.gitDirty = len(strings.TrimSpace(string(output))) > 0
	}
}

// View отрисовывает UI (Elm Architecture)
func (m Model) View() string {
	if !m.ready {
		return ""
	}

	if m.quitting {
		return ""
	}

	// Если показываем help overlay - рисуем его поверх основного UI
	if m.showingHelp {
		return m.renderWithHelpOverlay()
	}

	// Выбираем рендеринг в зависимости от режима
	switch m.config.UI.Mode {
	case config.UIModeClassic:
		return m.renderClassicMode()
	case config.UIModeWarp:
		return m.renderWarpMode()
	case config.UIModeCompact:
		return m.renderCompactMode()
	case config.UIModeChat:
		return m.renderChatMode()
	default:
		return m.renderClassicMode() // Fallback
	}
}

// renderClassicMode - классический режим (bash/pwsh)
// Текущий промпт + input ВНУТРИ viewport (как в настоящем bash/pwsh)
func (m Model) renderClassicMode() string {
	// Строим content с текущим промптом ВНУТРИ
	content := m.buildClassicViewportContent()
	m.viewport.SetContent(content)

	// Скроллим в конец только если autoScroll включен
	if m.autoScroll {
		m.viewport.GotoBottom()
	}

	// Возвращаем только viewport - всё внутри него!
	return m.viewport.View()
}

// buildClassicViewportContent строит содержимое viewport для Classic mode
// Включает историю + текущий промпт + текущий input (LIVE обновляется)
func (m Model) buildClassicViewportContent() string {
	var b strings.Builder

	// Вся история команд
	if len(m.output) > 0 {
		b.WriteString(strings.Join(m.output, "\n"))
		b.WriteString("\n")
	}

	// ТЕКУЩИЙ промпт + input (это "живая" строка)
	if m.executing {
		// Во время выполнения показываем spinner
		b.WriteString(m.executingSpinner.View())
		b.WriteString(" ")
		b.WriteString(m.styles.Executing.Render("Executing..."))
	} else {
		// Обычное состояние - промпт + ввод
		b.WriteString(m.renderPromptForHistoryANSI())
		b.WriteString(m.renderInputWithCursor())
		b.WriteString(m.renderHints())
	}

	return b.String()
}

// renderWarpMode - современный режим (Warp)
// Промпт сверху, вывод снизу с разделителем
func (m Model) renderWarpMode() string {
	var b strings.Builder

	// Prompt СВЕРХУ
	if !m.executing {
		b.WriteString(m.renderPromptForHistoryANSI())
	} else {
		b.WriteString(m.executingSpinner.View())
		b.WriteString(" ")
		b.WriteString(m.styles.Executing.Render("Executing..."))
		b.WriteString(" ")
	}

	// Input
	b.WriteString(m.renderInputWithCursor())

	// Hints
	b.WriteString(m.renderHints())

	b.WriteString("\n")

	// Разделитель
	b.WriteString(strings.Repeat("─", m.width))
	b.WriteString("\n")

	// Output history СНИЗУ
	b.WriteString(m.viewport.View())

	return b.String()
}

// renderCompactMode - компактный режим
// Минималистичный промпт, максимум места для вывода
func (m Model) renderCompactMode() string {
	var b strings.Builder

	// Output history СВЕРХУ
	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	// Executing indicator (компактный)
	if m.executing {
		b.WriteString(m.executingSpinner.View())
		b.WriteString(" ")
	}

	// Compact prompt
	if !m.executing {
		b.WriteString("$ ")
	}

	// Input
	b.WriteString(m.renderInputWithCursor())

	// Hints (компактные)
	b.WriteString(m.renderHints())

	return b.String()
}

// renderChatMode - чат режим (Telegram/ChatGPT)
// Ввод зафиксирован внизу, история прокручивается сверху
func (m Model) renderChatMode() string {
	var b strings.Builder

	// Output history СВЕРХУ (основная часть экрана)
	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	// Разделитель
	b.WriteString(strings.Repeat("─", m.width))
	b.WriteString("\n")

	// Executing indicator
	if m.executing {
		b.WriteString(m.executingSpinner.View())
		b.WriteString(" ")
		b.WriteString(m.styles.Executing.Render("Executing..."))
		b.WriteString(" ")
	} else {
		// Compact prompt для chat mode
		b.WriteString(m.styles.PromptArrow.Render("→ "))
	}

	// Input (зафиксирован внизу)
	b.WriteString(m.renderInputWithCursor())

	// Hints
	b.WriteString(m.renderHints())

	return b.String()
}

// renderInputWithCursor отрисовывает ввод с курсором и подсветкой
func (m Model) renderInputWithCursor() string {
	currentText := m.textarea.Value()
	if currentText != "" {
		// Применяем подсветку
		highlighted := m.applySyntaxHighlight(currentText)
		// Вставляем курсор в правильную позицию
		return m.insertCursorIntoHighlighted(highlighted, m.cursorPos)
	}

	// Пустой ввод - показываем блинкающий курсор
	return "\033[7m \033[0m"
}

// renderHints отрисовывает подсказки (completion, scroll indicator)
func (m Model) renderHints() string {
	var hints []string

	// Completion hint
	if m.completionActive && len(m.completions) > 1 {
		hint := fmt.Sprintf("[Tab: %d/%d]", m.completionIndex+1, len(m.completions))
		hints = append(hints, m.styles.CompletionHint.Render(hint))
	}

	// Scroll indicator
	if !m.autoScroll && m.viewport.ScrollPercent() < 0.99 {
		hints = append(hints, m.styles.CompletionHint.Render("[↑ scrolled]"))
	}

	if len(hints) > 0 {
		return " " + strings.Join(hints, " ")
	}

	return ""
}

// insertCursorIntoHighlighted вставляет курсор в highlighted текст с ANSI кодами
func (m Model) insertCursorIntoHighlighted(highlighted string, cursorPos int) string {
	if cursorPos < 0 {
		cursorPos = 0
	}

	var result strings.Builder
	visibleCount := 0
	inEscape := false
	runes := []rune(highlighted)
	cursorInserted := false

	for i := 0; i < len(runes); i++ {
		r := runes[i]

		// Обнаружение ANSI escape последовательности
		if r == '\033' {
			inEscape = true
			result.WriteRune(r)
			continue
		}

		if inEscape {
			result.WriteRune(r)
			if r == 'm' || r == 'H' || r == 'J' || r == 'K' {
				inEscape = false
			}
			continue
		}

		// Видимый символ
		if visibleCount == cursorPos && !cursorInserted {
			// Вставляем курсор перед этим символом
			result.WriteString("\033[7m") // Инверсное видео
			result.WriteRune(r)
			result.WriteString("\033[27m") // Отключаем инверсное видео
			cursorInserted = true
		} else {
			result.WriteRune(r)
		}

		visibleCount++
	}

	// Если курсор в конце
	if !cursorInserted && visibleCount == cursorPos {
		result.WriteString("\033[7m \033[0m")
	}

	return result.String()
}

// applySyntaxHighlight применяет простую bash подсветку БЕЗ Chroma
func (m Model) applySyntaxHighlight(text string) string {
	if text == "" {
		return ""
	}

	// Простая подсветка: разбиваем на токены по пробелам
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return text
	}

	var result strings.Builder

	for i, part := range parts {
		if i > 0 {
			result.WriteString(" ") // Пробел между токенами
		}

		if i == 0 {
			// Первое слово = КОМАНДА (ЖЁЛТЫЙ)
			result.WriteString("\033[1;33m") // Bright Yellow
			result.WriteString(part)
			result.WriteString("\033[0m")
		} else if strings.HasPrefix(part, "-") {
			// Опция (СЕРЫЙ)
			result.WriteString("\033[90m") // Dark Gray
			result.WriteString(part)
			result.WriteString("\033[0m")
		} else {
			// Аргумент (ЗЕЛЁНЫЙ)
			result.WriteString("\033[32m") // Green
			result.WriteString(part)
			result.WriteString("\033[0m")
		}
	}

	return result.String()
}

// renderPrompt отрисовывает prompt в стиле PowerShell/Git Bash
func (m Model) renderPrompt() string {
	var parts []string

	// Username@hostname (зеленый)
	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME")
	}
	hostname, _ := os.Hostname()

	userHost := m.styles.PromptUser.Render(fmt.Sprintf("%s@%s", username, hostname))
	parts = append(parts, userHost)

	// Путь (синий)
	workDir := m.currentSession.WorkingDirectory()
	displayPath := m.shortenPath(workDir)
	pathStr := m.styles.PromptPath.Render(displayPath)
	parts = append(parts, pathStr)

	// Git статус (если есть)
	if m.gitBranch != "" {
		gitStr := ""
		if m.gitDirty {
			gitStr = m.styles.PromptGitDirty.Render(fmt.Sprintf("(%s *)", m.gitBranch))
		} else {
			gitStr = m.styles.PromptGit.Render(fmt.Sprintf("(%s)", m.gitBranch))
		}
		parts = append(parts, gitStr)
	}

	prompt := strings.Join(parts, " ")

	// Стрелка (зеленая или красная в зависимости от exitCode)
	var arrow string
	if m.lastExitCode == 0 {
		arrow = m.styles.PromptArrow.Render(" $ ")
	} else {
		arrow = m.styles.PromptError.Render(fmt.Sprintf(" [%d] $ ", m.lastExitCode))
	}

	return prompt + arrow
}

// renderPromptForHistory отрисовывает prompt для истории (с lipgloss)
func (m Model) renderPromptForHistory() string {
	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME")
	}
	hostname, _ := os.Hostname()

	workDir := m.currentSession.WorkingDirectory()
	displayPath := m.shortenPath(workDir)

	var parts []string
	parts = append(parts, m.styles.PromptUser.Render(fmt.Sprintf("%s@%s", username, hostname)))
	parts = append(parts, m.styles.PromptPath.Render(displayPath))

	if m.gitBranch != "" {
		if m.gitDirty {
			parts = append(parts, m.styles.PromptGitDirty.Render(fmt.Sprintf("(%s *)", m.gitBranch)))
		} else {
			parts = append(parts, m.styles.PromptGit.Render(fmt.Sprintf("(%s)", m.gitBranch)))
		}
	}

	arrow := m.styles.PromptArrow.Render(" $ ")
	return strings.Join(parts, " ") + arrow
}

// renderPromptForHistoryANSI отрисовывает prompt для истории (только ANSI коды)
func (m Model) renderPromptForHistoryANSI() string {
	const (
		reset      = "\033[0m"
		boldGreen  = "\033[1;32m" // username@hostname
		blue       = "\033[34m"   // path
		purple     = "\033[35m"   // git clean
		boldYellow = "\033[1;33m" // git dirty
	)

	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME")
	}
	hostname, _ := os.Hostname()

	workDir := m.currentSession.WorkingDirectory()
	displayPath := m.shortenPath(workDir)

	var result strings.Builder

	// username@hostname (жирный зеленый)
	result.WriteString(boldGreen)
	result.WriteString(fmt.Sprintf("%s@%s", username, hostname))
	result.WriteString(reset)
	result.WriteString(" ")

	// path (синий)
	result.WriteString(blue)
	result.WriteString(displayPath)
	result.WriteString(reset)

	// git status
	if m.gitBranch != "" {
		result.WriteString(" ")
		if m.gitDirty {
			result.WriteString(boldYellow)
			result.WriteString(fmt.Sprintf("(%s *)", m.gitBranch))
			result.WriteString(reset)
		} else {
			result.WriteString(purple)
			result.WriteString(fmt.Sprintf("(%s)", m.gitBranch))
			result.WriteString(reset)
		}
	}

	// arrow (жирный зеленый)
	result.WriteString(" ")
	result.WriteString(boldGreen)
	result.WriteString("$")
	result.WriteString(reset)
	result.WriteString(" ")

	return result.String()
}

// renderWithHelpOverlay рисует help overlay поверх основного UI
func (m Model) renderWithHelpOverlay() string {
	// Создаем help overlay
	helpOverlay := m.renderHelpOverlay()

	// Размещаем overlay по центру экрана
	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		helpOverlay,
	)
}

// renderHelpOverlay создает модальное окно помощи
func (m Model) renderHelpOverlay() string {
	// Стиль для overlay box
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("12")). // Синий
		Padding(1, 2).
		Width(60).
		Background(lipgloss.Color("0")). // Черный фон
		Foreground(lipgloss.Color("15")) // Белый текст

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("11")). // Желтый
		Bold(true)

	sectionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("10")). // Зеленый
		Bold(true)

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("14")) // Cyan

	var content strings.Builder

	// Заголовок
	content.WriteString(titleStyle.Render("GoSh Keyboard Shortcuts"))
	content.WriteString("\n\n")

	// Navigation
	content.WriteString(sectionStyle.Render("Navigation:"))
	content.WriteString("\n")
	content.WriteString(keyStyle.Render("  ↑/↓       ") + " - Command history\n")
	content.WriteString(keyStyle.Render("  Tab       ") + " - Auto-complete\n")
	content.WriteString(keyStyle.Render("  PgUp/PgDn ") + " - Scroll output\n")
	content.WriteString("\n")

	// Input
	content.WriteString(sectionStyle.Render("Input:"))
	content.WriteString("\n")
	content.WriteString(keyStyle.Render("  Enter     ") + " - Execute command\n")
	content.WriteString(keyStyle.Render("  Alt+Enter ") + " - Multi-line input\n")
	content.WriteString(keyStyle.Render("  Ctrl+L    ") + " - Clear screen\n")
	content.WriteString("\n")

	// UI Modes (если разрешено переключение)
	if m.config.UI.AllowModeSwitching {
		content.WriteString(sectionStyle.Render("UI Modes:"))
		content.WriteString("\n")
		content.WriteString(keyStyle.Render("  Ctrl+F5   ") + " - Classic mode\n")
		content.WriteString(keyStyle.Render("  Ctrl+F6   ") + " - Warp mode\n")
		content.WriteString(keyStyle.Render("  Ctrl+F7   ") + " - Compact mode\n")
		content.WriteString(keyStyle.Render("  Ctrl+F8   ") + " - Chat mode\n")
		content.WriteString("\n")
	}

	// Help
	content.WriteString(sectionStyle.Render("Help:"))
	content.WriteString("\n")
	content.WriteString(keyStyle.Render("  F1 or ?   ") + " - This help\n")
	content.WriteString(keyStyle.Render("  help      ") + " - Built-in commands\n")
	content.WriteString(keyStyle.Render("  ESC       ") + " - Close this help\n")
	content.WriteString("\n")

	// Exit
	content.WriteString(sectionStyle.Render("Exit:"))
	content.WriteString("\n")
	content.WriteString(keyStyle.Render("  Ctrl+C/D  ") + " - Exit shell\n")
	content.WriteString(keyStyle.Render("  exit      ") + " - Exit shell\n")

	return boxStyle.Render(content.String())
}

// shortenPath сокращает путь для отображения
func (m Model) shortenPath(path string) string {
	home, _ := os.UserHomeDir()

	// Заменяем home на ~
	if strings.HasPrefix(path, home) {
		path = "~" + strings.TrimPrefix(path, home)
	}

	// Если путь слишком длинный, показываем только последние 3 компонента
	parts := strings.Split(path, string(filepath.Separator))
	if len(parts) > 3 && !strings.HasPrefix(path, "~") {
		path = ".../" + strings.Join(parts[len(parts)-2:], "/")
	}

	return path
}

// makeProfessionalStyles создает профессиональные стили как в PowerShell/Git Bash
func makeProfessionalStyles() Styles {
	return Styles{
		// Prompt - PowerShell/Bash inspired colors
		PromptUser: lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")). // Зеленый
			Bold(true),

		PromptPath: lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")), // Синий

		PromptGit: lipgloss.NewStyle().
			Foreground(lipgloss.Color("13")), // Фиолетовый

		PromptGitDirty: lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")). // Желтый
			Bold(true),

		PromptArrow: lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")). // Зеленый
			Bold(true),

		PromptError: lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")). // Красный
			Bold(true),

		// Output
		Output: lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")), // Белый

		OutputErr: lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")), // Красный

		// Executing
		Executing: lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")). // Синий
			Italic(true),

		// Completion hint
		CompletionHint: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")). // Серый
			Italic(true),

		// Syntax highlighting (базовые ANSI цвета для совместимости)
		SyntaxCommand: lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")). // Яркий желтый (команда ярко)
			Bold(true),

		SyntaxOption: lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")), // Темно-серый (опции тускло)

		SyntaxArg: lipgloss.NewStyle().
			Foreground(lipgloss.Color("7")), // Светло-серый (аргументы обычно)

		SyntaxString: lipgloss.NewStyle().
			Foreground(lipgloss.Color("14")), // Cyan (строки)
	}
}
