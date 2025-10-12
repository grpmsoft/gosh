package shared

// ExitCode represents command return code
type ExitCode int

const (
	ExitSuccess ExitCode = 0
	ExitFailure ExitCode = 1
	ExitError   ExitCode = 2
)

// IsSuccess checks if execution was successful
func (e ExitCode) IsSuccess() bool {
	return e == ExitSuccess
}

// Environment represents environment variables
type Environment map[string]string

// Clone creates a copy of the environment
func (e Environment) Clone() Environment {
	clone := make(Environment, len(e))
	for k, v := range e {
		clone[k] = v
	}
	return clone
}

// Set sets an environment variable
func (e Environment) Set(key, value string) {
	e[key] = value
}

// Get gets an environment variable
func (e Environment) Get(key string) (string, bool) {
	val, ok := e[key]
	return val, ok
}

// Unset removes an environment variable
func (e Environment) Unset(key string) {
	delete(e, key)
}

// ToSlice converts to a slice of strings in "KEY=VALUE" format
func (e Environment) ToSlice() []string {
	result := make([]string, 0, len(e))
	for k, v := range e {
		result = append(result, k+"="+v)
	}
	return result
}
