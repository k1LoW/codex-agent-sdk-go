package codex

import "fmt"

// SDKError is the base error for all codex-agent-sdk-go errors.
type SDKError struct {
	Message string
	Cause   error
}

func (e *SDKError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *SDKError) Unwrap() error { return e.Cause }

// CLINotFoundError is returned when the codex binary cannot be found.
type CLINotFoundError struct {
	SDKError
	CLIPath string
}

// CLIConnectionError is returned when the connection to the CLI fails.
type CLIConnectionError struct {
	SDKError
}

// ProcessError is returned when the codex process exits with an error.
type ProcessError struct {
	SDKError
	ExitCode int
	Stderr   string
}

// JSONDecodeError is returned when a JSON message cannot be decoded.
type JSONDecodeError struct {
	SDKError
	Line string
}

// RPCError is returned when the server responds with a JSON-RPC error.
type RPCError struct {
	SDKError
	Code int
	Data any
}

// TurnError is returned when a turn fails.
type TurnError struct {
	SDKError
	TurnID string
}
