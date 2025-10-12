package history_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/grpmsoft/gosh/internal/domain/history"
	historyInfra "github.com/grpmsoft/gosh/internal/infrastructure/history"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFileHistoryRepository_Save проверяет сохранение истории в файл
func TestFileHistoryRepository_Save(t *testing.T) {
	t.Run("saves history to file", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test_history.txt")

		repo := historyInfra.NewFileHistoryRepository(filePath)

		h := history.NewHistory(history.DefaultConfig())
		h.Add("git status")
		h.Add("git commit")
		h.Add("git push")

		err := repo.Save(h)
		require.NoError(t, err)

		// Проверяем что файл создан
		assert.FileExists(t, filePath)

		// Проверяем содержимое
		content, err := os.ReadFile(filePath)
		require.NoError(t, err)

		// Файл должен содержать команды в хронологическом порядке
		expected := "git status\ngit commit\ngit push\n"
		assert.Equal(t, expected, string(content))
	})

	t.Run("creates parent directories if needed", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "nested", "dir", "history.txt")

		repo := historyInfra.NewFileHistoryRepository(filePath)

		h := history.NewHistory(history.DefaultConfig())
		h.Add("test command")

		err := repo.Save(h)
		require.NoError(t, err)

		assert.FileExists(t, filePath)
	})

	t.Run("overwrites existing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "history.txt")

		// Создаем файл с начальным содержимым
		err := os.WriteFile(filePath, []byte("old command 1\nold command 2\n"), 0644)
		require.NoError(t, err)

		repo := historyInfra.NewFileHistoryRepository(filePath)

		h := history.NewHistory(history.DefaultConfig())
		h.Add("new command")

		err = repo.Save(h)
		require.NoError(t, err)

		// Проверяем что старые команды удалены
		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, "new command\n", string(content))
	})

	t.Run("saves empty history as empty file", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "history.txt")

		repo := historyInfra.NewFileHistoryRepository(filePath)

		h := history.NewHistory(history.DefaultConfig())

		err := repo.Save(h)
		require.NoError(t, err)

		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Empty(t, string(content))
	})

	t.Run("handles special characters in commands", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "history.txt")

		repo := historyInfra.NewFileHistoryRepository(filePath)

		h := history.NewHistory(history.DefaultConfig())
		h.Add("echo 'Hello World'")
		h.Add("grep \"test\" file.txt")
		h.Add("cmd with\ttab")

		err := repo.Save(h)
		require.NoError(t, err)

		content, err := os.ReadFile(filePath)
		require.NoError(t, err)

		lines := string(content)
		assert.Contains(t, lines, "echo 'Hello World'")
		assert.Contains(t, lines, "grep \"test\" file.txt")
		assert.Contains(t, lines, "cmd with\ttab")
	})
}

// TestFileHistoryRepository_Load проверяет загрузку истории из файла
func TestFileHistoryRepository_Load(t *testing.T) {
	t.Run("loads history from file", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "history.txt")

		// Создаем файл с тестовыми данными
		content := "git status\ngit commit\ngit push\n"
		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)

		repo := historyInfra.NewFileHistoryRepository(filePath)

		h := history.NewHistory(history.DefaultConfig())
		err = repo.Load(h)
		require.NoError(t, err)

		assert.Equal(t, 3, h.Size())

		recent := h.GetRecent(3)
		assert.Equal(t, "git push", recent[0])
		assert.Equal(t, "git commit", recent[1])
		assert.Equal(t, "git status", recent[2])
	})

	t.Run("loads empty file as empty history", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "history.txt")

		err := os.WriteFile(filePath, []byte(""), 0644)
		require.NoError(t, err)

		repo := historyInfra.NewFileHistoryRepository(filePath)

		h := history.NewHistory(history.DefaultConfig())
		err = repo.Load(h)
		require.NoError(t, err)

		assert.Equal(t, 0, h.Size())
	})

	t.Run("returns no error if file does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "nonexistent.txt")

		repo := historyInfra.NewFileHistoryRepository(filePath)

		h := history.NewHistory(history.DefaultConfig())
		err := repo.Load(h)

		// Не ошибка - просто пустая история
		assert.NoError(t, err)
		assert.Equal(t, 0, h.Size())
	})

	t.Run("skips empty lines and whitespace", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "history.txt")

		content := "cmd1\n\n  \ncmd2\n\t\ncmd3\n"
		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)

		repo := historyInfra.NewFileHistoryRepository(filePath)

		h := history.NewHistory(history.DefaultConfig())
		err = repo.Load(h)
		require.NoError(t, err)

		assert.Equal(t, 3, h.Size())
		recent := h.GetRecent(3)
		assert.Equal(t, "cmd3", recent[0])
		assert.Equal(t, "cmd2", recent[1])
		assert.Equal(t, "cmd1", recent[2])
	})

	t.Run("handles large history files", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "history.txt")

		// Создаем файл с 10000 командами
		content := ""
		for i := 1; i <= 10000; i++ {
			content += "command " + string(rune(i)) + "\n"
		}
		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)

		repo := historyInfra.NewFileHistoryRepository(filePath)

		cfg := history.Config{
			MaxSize:          5000, // Ограничение истории
			DeduplicateAdded: false,
		}
		h := history.NewHistory(cfg)

		err = repo.Load(h)
		require.NoError(t, err)

		// Должны загрузиться только последние 5000
		assert.Equal(t, 5000, h.Size())
	})

	t.Run("preserves special characters", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "history.txt")

		content := "echo 'hello'\ngrep \"test\"\ncmd\twith\ttab\n"
		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)

		repo := historyInfra.NewFileHistoryRepository(filePath)

		h := history.NewHistory(history.DefaultConfig())
		err = repo.Load(h)
		require.NoError(t, err)

		slice := h.ToSlice()
		assert.Equal(t, "echo 'hello'", slice[0])
		assert.Equal(t, "grep \"test\"", slice[1])
		assert.Equal(t, "cmd\twith\ttab", slice[2])
	})
}

// TestFileHistoryRepository_SaveAndLoad проверяет круговорот сохранения/загрузки
func TestFileHistoryRepository_SaveAndLoad(t *testing.T) {
	t.Run("round trip preserves history", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "history.txt")

		repo := historyInfra.NewFileHistoryRepository(filePath)

		// Создаем и сохраняем историю
		h1 := history.NewHistory(history.DefaultConfig())
		h1.Add("git status")
		h1.Add("git commit -m 'test'")
		h1.Add("git push origin main")

		err := repo.Save(h1)
		require.NoError(t, err)

		// Загружаем в новый объект
		h2 := history.NewHistory(history.DefaultConfig())
		err = repo.Load(h2)
		require.NoError(t, err)

		// Проверяем идентичность
		assert.Equal(t, h1.Size(), h2.Size())
		assert.Equal(t, h1.ToSlice(), h2.ToSlice())
	})
}

// TestFileHistoryRepository_ExpandTilde проверяет раскрытие ~ в пути
func TestFileHistoryRepository_ExpandTilde(t *testing.T) {
	t.Run("expands tilde to home directory", func(t *testing.T) {
		// Эта функциональность должна быть в репозитории
		repo := historyInfra.NewFileHistoryRepository("~/.gosh_history")

		// Путь должен быть раскрыт
		actualPath := repo.FilePath()
		assert.NotContains(t, actualPath, "~")

		homeDir, err := os.UserHomeDir()
		require.NoError(t, err)
		assert.Contains(t, actualPath, homeDir)
	})

	t.Run("leaves absolute paths unchanged", func(t *testing.T) {
		path := "/tmp/history.txt"
		repo := historyInfra.NewFileHistoryRepository(path)

		assert.Equal(t, path, repo.FilePath())
	})
}

// TestFileHistoryRepository_Concurrency проверяет потокобезопасность
func TestFileHistoryRepository_Concurrency(t *testing.T) {
	t.Run("handles concurrent saves", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "history.txt")

		repo := historyInfra.NewFileHistoryRepository(filePath)

		h := history.NewHistory(history.DefaultConfig())
		h.Add("test command")

		// Запускаем 10 одновременных сохранений
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				defer func() { done <- true }()
				err := repo.Save(h)
				assert.NoError(t, err)
			}()
		}

		// Ждем завершения всех
		for i := 0; i < 10; i++ {
			<-done
		}
		close(done)

		// Файл должен существовать и быть валидным
		assert.FileExists(t, filePath)

		// Даем время на закрытие файловых дескрипторов на Windows
		// Это предотвращает ошибку cleanup в TempDir
		time.Sleep(10 * time.Millisecond)
	})
}

// TestFileHistoryRepository_Append проверяет добавление команд в конец файла
func TestFileHistoryRepository_Append(t *testing.T) {
	t.Run("appends single command to new file", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "history.txt")

		repo := historyInfra.NewFileHistoryRepository(filePath)

		err := repo.Append("git status")
		require.NoError(t, err)

		// Проверяем что файл создан
		assert.FileExists(t, filePath)

		// Проверяем содержимое
		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, "git status\n", string(content))
	})

	t.Run("appends multiple commands", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "history.txt")

		repo := historyInfra.NewFileHistoryRepository(filePath)

		err := repo.Append("git status")
		require.NoError(t, err)

		err = repo.Append("git commit")
		require.NoError(t, err)

		err = repo.Append("git push")
		require.NoError(t, err)

		// Проверяем что все команды добавлены в хронологическом порядке
		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, "git status\ngit commit\ngit push\n", string(content))
	})

	t.Run("appends to existing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "history.txt")

		// Создаем файл с начальным содержимым
		err := os.WriteFile(filePath, []byte("old command\n"), 0644)
		require.NoError(t, err)

		repo := historyInfra.NewFileHistoryRepository(filePath)

		err = repo.Append("new command")
		require.NoError(t, err)

		// Проверяем что старая команда сохранилась
		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, "old command\nnew command\n", string(content))
	})

	t.Run("skips empty commands", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "history.txt")

		repo := historyInfra.NewFileHistoryRepository(filePath)

		err := repo.Append("")
		require.NoError(t, err)

		err = repo.Append("  ")
		require.NoError(t, err)

		err = repo.Append("\t")
		require.NoError(t, err)

		// Файл не должен быть создан для пустых команд
		_, err = os.Stat(filePath)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("handles special characters", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "history.txt")

		repo := historyInfra.NewFileHistoryRepository(filePath)

		err := repo.Append("echo 'Hello World'")
		require.NoError(t, err)

		err = repo.Append("grep \"test\" file.txt")
		require.NoError(t, err)

		content, err := os.ReadFile(filePath)
		require.NoError(t, err)

		assert.Contains(t, string(content), "echo 'Hello World'")
		assert.Contains(t, string(content), "grep \"test\" file.txt")
	})

	t.Run("creates parent directories if needed", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "nested", "dir", "history.txt")

		repo := historyInfra.NewFileHistoryRepository(filePath)

		err := repo.Append("test command")
		require.NoError(t, err)

		assert.FileExists(t, filePath)
	})

	t.Run("handles concurrent appends from multiple goroutines", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "history.txt")

		repo := historyInfra.NewFileHistoryRepository(filePath)

		// Запускаем 100 одновременных записей
		done := make(chan bool, 100)
		for i := 0; i < 100; i++ {
			go func(n int) {
				defer func() { done <- true }()
				err := repo.Append("command " + string(rune(48+n%10)))
				assert.NoError(t, err)
			}(i)
		}

		// Ждем завершения всех
		for i := 0; i < 100; i++ {
			<-done
		}
		close(done)

		// Проверяем что файл существует
		assert.FileExists(t, filePath)

		// Проверяем что все команды записаны
		content, err := os.ReadFile(filePath)
		require.NoError(t, err)

		// Должно быть 100 строк
		lines := 0
		for _, c := range string(content) {
			if c == '\n' {
				lines++
			}
		}
		assert.Equal(t, 100, lines, "Should have 100 commands written")

		// Даем время на закрытие файловых дескрипторов на Windows
		time.Sleep(10 * time.Millisecond)
	})
}

// TestFileHistoryRepository_AppendAndLoad проверяет что Append работает с Load
func TestFileHistoryRepository_AppendAndLoad(t *testing.T) {
	t.Run("load after append preserves all commands", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "history.txt")

		repo := historyInfra.NewFileHistoryRepository(filePath)

		// Append несколько команд
		repo.Append("cmd1")
		repo.Append("cmd2")
		repo.Append("cmd3")

		// Load в историю
		h := history.NewHistory(history.DefaultConfig())
		err := repo.Load(h)
		require.NoError(t, err)

		// Проверяем что все команды загружены
		assert.Equal(t, 3, h.Size())
		slice := h.ToSlice()
		assert.Equal(t, "cmd1", slice[0])
		assert.Equal(t, "cmd2", slice[1])
		assert.Equal(t, "cmd3", slice[2])
	})

	t.Run("simulates multi-terminal scenario", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "history.txt")

		// Terminal 1 adds commands
		repo1 := historyInfra.NewFileHistoryRepository(filePath)
		repo1.Append("terminal1-cmd1")
		repo1.Append("terminal1-cmd2")

		// Terminal 2 adds commands
		repo2 := historyInfra.NewFileHistoryRepository(filePath)
		repo2.Append("terminal2-cmd1")
		repo2.Append("terminal2-cmd2")

		// Terminal 3 loads history
		repo3 := historyInfra.NewFileHistoryRepository(filePath)
		h := history.NewHistory(history.DefaultConfig())
		err := repo3.Load(h)
		require.NoError(t, err)

		// Should have all 4 commands
		assert.Equal(t, 4, h.Size())

		slice := h.ToSlice()
		// Commands should be in append order (oldest first)
		assert.Equal(t, "terminal1-cmd1", slice[0])
		assert.Equal(t, "terminal1-cmd2", slice[1])
		assert.Equal(t, "terminal2-cmd1", slice[2])
		assert.Equal(t, "terminal2-cmd2", slice[3])
	})
}
