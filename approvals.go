package codex

import "context"

// ApprovalDecision is the decision for an approval request.
type ApprovalDecision string

const (
	DecisionAccept           ApprovalDecision = "accept"
	DecisionAcceptForSession ApprovalDecision = "acceptForSession"
	DecisionDecline          ApprovalDecision = "decline"
	DecisionCancel           ApprovalDecision = "cancel"
)

// CommandApprovalRequest is passed to the OnCommandApproval callback.
type CommandApprovalRequest struct {
	ThreadID string
	TurnID   string
	ItemID   string
	Command  string
	CWD      string
	Reason   string
	// Raw contains the full params for advanced use.
	Raw map[string]any
}

// FileChangeApprovalRequest is passed to the OnFileChangeApproval callback.
type FileChangeApprovalRequest struct {
	ThreadID string
	TurnID   string
	ItemID   string
	Reason   string
	// Raw contains the full params for advanced use.
	Raw map[string]any
}

// PermissionApprovalRequest is passed to the OnPermissionApproval callback.
type PermissionApprovalRequest struct {
	ThreadID string
	TurnID   string
	// Raw contains the full params for advanced use.
	Raw map[string]any
}

// ToolCallRequest is passed to the OnToolCall callback.
type ToolCallRequest struct {
	ThreadID  string
	TurnID    string
	CallID    string
	Tool      string
	Arguments map[string]any
}

// ToolCallResponse is returned from the OnToolCall callback.
type ToolCallResponse struct {
	ContentItems []map[string]any
	Success      bool
}

// UserInputRequest is passed to the OnUserInput callback.
type UserInputRequest struct {
	ThreadID  string
	TurnID    string
	ItemID    string
	Questions []map[string]any
}

// ElicitationRequest is passed to the OnElicitation callback.
type ElicitationRequest struct {
	ThreadID   string
	TurnID     string
	ServerName string
	// Raw contains the full params for advanced use.
	Raw map[string]any
}

// ElicitationAction is the action for an elicitation response.
type ElicitationAction string

const (
	ElicitationAccept  ElicitationAction = "accept"
	ElicitationDecline ElicitationAction = "decline"
	ElicitationCancel  ElicitationAction = "cancel"
)

// ElicitationResponse is returned from the OnElicitation callback.
type ElicitationResponse struct {
	Action   ElicitationAction
	Response map[string]any
}

// Callback type signatures.

// OnCommandApprovalFunc is called when the server requests command execution approval.
type OnCommandApprovalFunc func(ctx context.Context, req CommandApprovalRequest) (ApprovalDecision, error)

// OnFileChangeApprovalFunc is called when the server requests file change approval.
type OnFileChangeApprovalFunc func(ctx context.Context, req FileChangeApprovalRequest) (ApprovalDecision, error)

// OnPermissionApprovalFunc is called when the server requests additional permission approval.
type OnPermissionApprovalFunc func(ctx context.Context, req PermissionApprovalRequest) (ApprovalDecision, error)

// OnToolCallFunc is called when the server requests a dynamic tool call.
type OnToolCallFunc func(ctx context.Context, req ToolCallRequest) (ToolCallResponse, error)

// OnUserInputFunc is called when the server requests user input.
type OnUserInputFunc func(ctx context.Context, req UserInputRequest) (map[string]string, error)

// OnElicitationFunc is called when the server requests MCP elicitation input.
type OnElicitationFunc func(ctx context.Context, req ElicitationRequest) (ElicitationResponse, error)
