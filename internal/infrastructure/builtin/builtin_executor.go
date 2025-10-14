// Package builtin provides adapters for executing shell builtin commands.
package builtin

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/grpmsoft/gosh/internal/application/ports"
	"github.com/grpmsoft/gosh/internal/domain/builtins"
	"github.com/grpmsoft/gosh/internal/domain/command"
	"github.com/grpmsoft/gosh/internal/domain/session"
	"github.com/grpmsoft/gosh/internal/domain/shared"
)

// Executor - executor for builtin commands.
type Executor struct {
	fs     ports.FileSystem
	logger *slog.Logger
	stdout io.Writer
	stderr io.Writer
}

// NewExecutor creates a new executor for builtin commands.
func NewExecutor(fs ports.FileSystem, logger *slog.Logger) *Executor {
	return &Executor{
		fs:     fs,
		logger: logger,
		stdout: os.Stdout,
		stderr: os.Stderr,
	}
}

// SetOutput sets output streams.
func (b *Executor) SetOutput(stdout, stderr io.Writer) {
	b.stdout = stdout
	b.stderr = stderr
}

// CanExecute checks if the executor can execute the command.
func (b *Executor) CanExecute(cmd *command.Command) bool {
	return cmd.IsBuiltin()
}

// Execute executes a builtin command and returns captured output.
func (b *Executor) Execute(
	_ context.Context,
	cmd *command.Command,
	sess *session.Session,
) (stdout, stderr string, err error) {
	b.logger.Debug("executing builtin command",
		"command", cmd.Name(),
	)

	// Create buffers to capture output
	var stdoutBuf, stderrBuf bytes.Buffer

	// Temporarily replace stdout/stderr with buffers
	oldStdout, oldStderr := b.stdout, b.stderr
	b.stdout = &stdoutBuf
	b.stderr = &stderrBuf

	// Restore original stdout/stderr after execution
	defer func() {
		b.stdout = oldStdout
		b.stderr = oldStderr
	}()

	// Execute the command
	var execErr error
	switch cmd.Name() {
	case "cd":
		execErr = b.cd(cmd, sess)
	case "pwd":
		execErr = b.pwd(sess)
	case "echo":
		execErr = b.echo(cmd)
	case "exit":
		execErr = b.exit(cmd)
	case "export":
		execErr = b.export(cmd, sess)
	case "unset":
		execErr = b.unset(cmd, sess)
	case "env":
		execErr = b.env(sess)
	case "alias":
		execErr = b.alias(cmd, sess)
	case "unalias":
		execErr = b.unalias(cmd, sess)
	case "type":
		execErr = b.typeCmd(cmd, sess)
	case "help":
		execErr = b.help()
	case "jobs":
		execErr = b.jobs(sess)
	case "fg":
		execErr = b.fg(cmd, sess)
	case "bg":
		execErr = b.bg(cmd, sess)
	default:
		execErr = shared.NewDomainError(
			"Execute",
			shared.ErrCommandNotFound,
			fmt.Sprintf("builtin command not implemented: %s", cmd.Name()),
		)
	}

	// Return captured output
	return stdoutBuf.String(), stderrBuf.String(), execErr
}

// cd - change working directory (delegates to domain).
func (b *Executor) cd(cmd *command.Command, sess *session.Session) error {
	cdCmd, err := builtins.NewCdCommand(cmd.Args(), sess)
	if err != nil {
		return err
	}
	return cdCmd.Execute()
}

// pwd - print working directory (delegates to domain).
func (b *Executor) pwd(sess *session.Session) error {
	pwdCmd, err := builtins.NewPwdCommand(sess, b.stdout)
	if err != nil {
		return err
	}
	return pwdCmd.Execute()
}

// echo - print arguments.
func (b *Executor) echo(cmd *command.Command) error {
	args := cmd.Args()
	output := strings.Join(args, " ")
	if _, err := fmt.Fprintln(b.stdout, output); err != nil {
		return shared.NewDomainError("echo", shared.ErrBuiltinFailed, err.Error())
	}
	return nil
}

// exit - exit shell.
//
//nolint:unparam // error always nil because os.Exit terminates process
func (b *Executor) exit(cmd *command.Command) error {
	// TODO: Properly handle exit codes
	exitCode := 0
	if len(cmd.Args()) > 0 {
		// Parse exit code from argument
		_, _ = fmt.Sscanf(cmd.Args()[0], "%d", &exitCode)
	}
	os.Exit(exitCode)
	return nil // unreachable but required for function signature
}

// export - export environment variable (delegates to domain).
func (b *Executor) export(cmd *command.Command, sess *session.Session) error {
	exportCmd, err := builtins.NewExportCommand(cmd.Args(), sess, b.stdout)
	if err != nil {
		return err
	}
	return exportCmd.Execute()
}

// unset - unset environment variable (delegates to domain).
func (b *Executor) unset(cmd *command.Command, sess *session.Session) error {
	unsetCmd, err := builtins.NewUnsetCommand(cmd.Args(), sess)
	if err != nil {
		return err
	}
	return unsetCmd.Execute()
}

// env - print all environment variables.
func (b *Executor) env(sess *session.Session) error {
	env := sess.Environment()
	for key, value := range env {
		if _, err := fmt.Fprintf(b.stdout, "%s=%s\n", key, value); err != nil {
			return shared.NewDomainError("env", shared.ErrBuiltinFailed, err.Error())
		}
	}
	return nil
}

// alias - manage aliases (delegates to domain).
func (b *Executor) alias(cmd *command.Command, sess *session.Session) error {
	aliasCmd, err := builtins.NewAliasCommand(cmd.Args(), sess, b.stdout)
	if err != nil {
		return err
	}
	return aliasCmd.Execute()
}

// unalias - remove aliases (delegates to domain).
func (b *Executor) unalias(cmd *command.Command, sess *session.Session) error {
	unaliasCmd, err := builtins.NewUnaliasCommand(cmd.Args(), sess)
	if err != nil {
		return err
	}
	return unaliasCmd.Execute()
}

// typeCmd - print command type (delegates to domain).
func (b *Executor) typeCmd(cmd *command.Command, sess *session.Session) error {
	typeCmd, err := builtins.NewTypeCommand(cmd.Args(), sess, b.stdout)
	if err != nil {
		return err
	}
	return typeCmd.Execute()
}

// jobs - list background jobs (delegates to domain).
func (b *Executor) jobs(sess *session.Session) error {
	jobsCmd, err := builtins.NewJobsCommand(sess, b.stdout)
	if err != nil {
		return err
	}
	return jobsCmd.Execute()
}

// fg - bring job to foreground (delegates to domain).
func (b *Executor) fg(cmd *command.Command, sess *session.Session) error {
	fgCmd, err := builtins.NewFgCommand(cmd.Args(), sess)
	if err != nil {
		return err
	}
	return fgCmd.Execute()
}

// bg - send job to background (delegates to domain).
func (b *Executor) bg(cmd *command.Command, sess *session.Session) error {
	bgCmd, err := builtins.NewBgCommand(cmd.Args(), sess)
	if err != nil {
		return err
	}
	return bgCmd.Execute()
}

// help - print help message.
func (b *Executor) help() error {
	help := `
gosh - Go Shell
Available builtin commands:

  cd [dir]          Change working directory (supports ~, cd -)
  pwd               Print working directory
  echo [args...]    Print arguments
  exit [code]       Exit shell with optional exit code
  export KEY=VALUE  Set environment variable
  unset KEY         Unset environment variable
  env               Print all environment variables
  alias [name=cmd]  Create or list aliases
  unalias name      Remove alias(es)
  type [cmd]        Show command type (builtin/alias/external)
  jobs              List background jobs
  fg [%n]           Bring job to foreground (default: most recent)
  bg [%n]           Resume stopped job in background (default: most recent)
  help              Show this help message

Use external commands by typing their name directly.
Pipeline commands with | (pipe operator).
Run commands in background with & (ampersand).
`
	if _, err := fmt.Fprintln(b.stdout, help); err != nil {
		return shared.NewDomainError("help", shared.ErrBuiltinFailed, err.Error())
	}
	return nil
}
