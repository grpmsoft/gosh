// Package command provides domain models for shell commands and their execution semantics.
package command

// BuiltinCommands list of built-in commands.
var BuiltinCommands = map[string]bool{
	"cd":      true,
	"pwd":     true,
	"echo":    true,
	"exit":    true,
	"export":  true,
	"unset":   true,
	"env":     true,
	"set":     true,
	"alias":   true,
	"unalias": true,
	"type":    true,
	"help":    true,
	"jobs":    true, // List background jobs
	"fg":      true, // Bring job to foreground
	"bg":      true, // Send job to background
}

// IsBuiltinCommand checks if a command is built-in.
func IsBuiltinCommand(name string) bool {
	return BuiltinCommands[name]
}

// GetCommandType determines the command type by name.
func GetCommandType(name string) Type {
	if IsBuiltinCommand(name) {
		return TypeBuiltin
	}
	return TypeExternal
}
