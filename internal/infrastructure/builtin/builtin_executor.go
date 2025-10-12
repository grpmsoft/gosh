package builtin

import (
	"context"
	"fmt"
	"github.com/grpmsoft/gosh/internal/application/ports"
	"github.com/grpmsoft/gosh/internal/domain/builtins"
	"github.com/grpmsoft/gosh/internal/domain/command"
	"github.com/grpmsoft/gosh/internal/domain/session"
	"github.com/grpmsoft/gosh/internal/domain/shared"
	"io"
	"log/slog"
	"os"
	"strings"
)

// BuiltinExecutor - executor for builtin commands
type BuiltinExecutor struct {
	fs     ports.FileSystem
	logger *slog.Logger
	stdout io.Writer
	stderr io.Writer
}

// NewBuiltinExecutor creates a new executor for builtin commands
func NewBuiltinExecutor(fs ports.FileSystem, logger *slog.Logger) *BuiltinExecutor {
	return &BuiltinExecutor{
		fs:     fs,
		logger: logger,
		stdout: os.Stdout,
		stderr: os.Stderr,
	}
}

// SetOutput sets output streams
func (b *BuiltinExecutor) SetOutput(stdout, stderr io.Writer) {
	b.stdout = stdout
	b.stderr = stderr
}

// CanExecute checks if the executor can execute the command
func (b *BuiltinExecutor) CanExecute(cmd *command.Command) bool {
	return cmd.IsBuiltin()
}

// Execute executes a builtin command
func (b *BuiltinExecutor) Execute(
	ctx context.Context,
	cmd *command.Command,
	sess *session.Session,
) error {
	b.logger.Debug("executing builtin command",
		"command", cmd.Name(),
	)

	switch cmd.Name() {
	case "cd":
		return b.cd(cmd, sess)
	case "pwd":
		return b.pwd(sess)
	case "echo":
		return b.echo(cmd)
	case "exit":
		return b.exit(cmd)
	case "export":
		return b.export(cmd, sess)
	case "unset":
		return b.unset(cmd, sess)
	case "env":
		return b.env(sess)
	case "alias":
		return b.alias(cmd, sess)
	case "unalias":
		return b.unalias(cmd, sess)
	case "type":
		return b.typeCmd(cmd, sess)
	case "help":
		return b.help()
	case "jobs":
		return b.jobs(sess)
	case "fg":
		return b.fg(cmd, sess)
	case "bg":
		return b.bg(cmd, sess)
	default:
		return shared.NewDomainError(
			"Execute",
			shared.ErrCommandNotFound,
			fmt.Sprintf("builtin command not implemented: %s", cmd.Name()),
		)
	}
}

// cd - change working directory (delegates to domain)
func (b *BuiltinExecutor) cd(cmd *command.Command, sess *session.Session) error {
	cdCmd, err := builtins.NewCdCommand(cmd.Args(), sess)
	if err != nil {
		return err
	}
	return cdCmd.Execute()
}

// pwd - print working directory (delegates to domain)
func (b *BuiltinExecutor) pwd(sess *session.Session) error {
	pwdCmd, err := builtins.NewPwdCommand(sess, b.stdout)
	if err != nil {
		return err
	}
	return pwdCmd.Execute()
}

// echo - print arguments
func (b *BuiltinExecutor) echo(cmd *command.Command) error {
	args := cmd.Args()
	output := strings.Join(args, " ")
	_, _ = fmt.Fprintln(b.stdout, output)
	return nil
}

// exit - exit shell
func (b *BuiltinExecutor) exit(cmd *command.Command) error {
	// TODO: Properly handle exit codes
	exitCode := 0
	if len(cmd.Args()) > 0 {
		// Parse exit code from argument
		_, _ = fmt.Sscanf(cmd.Args()[0], "%d", &exitCode)
	}
	os.Exit(exitCode)
	return nil
}

// export - export environment variable (delegates to domain)
func (b *BuiltinExecutor) export(cmd *command.Command, sess *session.Session) error {
	exportCmd, err := builtins.NewExportCommand(cmd.Args(), sess, b.stdout)
	if err != nil {
		return err
	}
	return exportCmd.Execute()
}

// unset - unset environment variable (delegates to domain)
func (b *BuiltinExecutor) unset(cmd *command.Command, sess *session.Session) error {
	unsetCmd, err := builtins.NewUnsetCommand(cmd.Args(), sess)
	if err != nil {
		return err
	}
	return unsetCmd.Execute()
}

// env - print all environment variables
func (b *BuiltinExecutor) env(sess *session.Session) error {
	env := sess.Environment()
	for key, value := range env {
		_, _ = fmt.Fprintf(b.stdout, "%s=%s\n", key, value)
	}
	return nil
}

// alias - manage aliases (delegates to domain)
func (b *BuiltinExecutor) alias(cmd *command.Command, sess *session.Session) error {
	aliasCmd, err := builtins.NewAliasCommand(cmd.Args(), sess, b.stdout)
	if err != nil {
		return err
	}
	return aliasCmd.Execute()
}

// unalias - remove aliases (delegates to domain)
func (b *BuiltinExecutor) unalias(cmd *command.Command, sess *session.Session) error {
	unaliasCmd, err := builtins.NewUnaliasCommand(cmd.Args(), sess)
	if err != nil {
		return err
	}
	return unaliasCmd.Execute()
}

// typeCmd - print command type (delegates to domain)
func (b *BuiltinExecutor) typeCmd(cmd *command.Command, sess *session.Session) error {
	typeCmd, err := builtins.NewTypeCommand(cmd.Args(), sess, b.stdout)
	if err != nil {
		return err
	}
	return typeCmd.Execute()
}

// jobs - list background jobs (delegates to domain)
func (b *BuiltinExecutor) jobs(sess *session.Session) error {
	jobsCmd, err := builtins.NewJobsCommand(sess, b.stdout)
	if err != nil {
		return err
	}
	return jobsCmd.Execute()
}

// fg - bring job to foreground (delegates to domain)
func (b *BuiltinExecutor) fg(cmd *command.Command, sess *session.Session) error {
	fgCmd, err := builtins.NewFgCommand(cmd.Args(), sess)
	if err != nil {
		return err
	}
	return fgCmd.Execute()
}

// bg - send job to background (delegates to domain)
func (b *BuiltinExecutor) bg(cmd *command.Command, sess *session.Session) error {
	bgCmd, err := builtins.NewBgCommand(cmd.Args(), sess)
	if err != nil {
		return err
	}
	return bgCmd.Execute()
}

// help - print help message
func (b *BuiltinExecutor) help() error {
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
	_, _ = fmt.Fprintln(b.stdout, help)
	return nil
}
