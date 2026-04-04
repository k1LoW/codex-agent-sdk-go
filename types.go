package codex

// Event is the interface implemented by all server notifications.
type Event interface { //nostyle:ifacenames // sealed marker interface, not a behavioral -er interface
	eventMethod() string
}

// ThreadStartedEvent is emitted when a new thread begins.
type ThreadStartedEvent struct {
	ThreadID string
	Thread   *Thread
}

func (*ThreadStartedEvent) eventMethod() string { return "thread/started" }

// TurnStartedEvent is emitted when a turn begins processing.
type TurnStartedEvent struct {
	ThreadID string
	Turn     *Turn
}

func (*TurnStartedEvent) eventMethod() string { return "turn/started" }

// TurnCompletedEvent is emitted when a turn finishes.
type TurnCompletedEvent struct {
	ThreadID string
	Turn     *Turn
}

func (*TurnCompletedEvent) eventMethod() string { return "turn/completed" }

// ItemStartedEvent is emitted when a new item begins.
type ItemStartedEvent struct {
	ThreadID string
	TurnID   string
	Item     ThreadItem
}

func (*ItemStartedEvent) eventMethod() string { return "item/started" }

// ItemCompletedEvent signals an item reached a terminal state.
type ItemCompletedEvent struct {
	ThreadID string
	TurnID   string
	Item     ThreadItem
}

func (*ItemCompletedEvent) eventMethod() string { return "item/completed" }

// AgentMessageDeltaEvent is emitted for streaming assistant message text.
type AgentMessageDeltaEvent struct {
	ThreadID string
	TurnID   string
	ItemID   string
	Delta    string
}

func (*AgentMessageDeltaEvent) eventMethod() string { return "item/agentMessage/delta" }

// CommandOutputDeltaEvent is emitted for streaming command execution output.
type CommandOutputDeltaEvent struct {
	ThreadID string
	TurnID   string
	ItemID   string
	Delta    string
}

func (*CommandOutputDeltaEvent) eventMethod() string { return "item/commandExecution/outputDelta" }

// ReasoningDeltaEvent is emitted for streaming reasoning text.
type ReasoningDeltaEvent struct {
	ThreadID     string
	TurnID       string
	ItemID       string
	Delta        string
	ContentIndex int
}

func (*ReasoningDeltaEvent) eventMethod() string { return "item/reasoning/textDelta" }

// ErrorEvent represents an error notification from the server.
type ErrorEvent struct {
	ThreadID string
	TurnID   string
	Code     int
	Message  string
	Details  string
}

func (*ErrorEvent) eventMethod() string { return "error" }

// ThreadItem is the interface implemented by all item types.
type ThreadItem interface {
	itemType() string //nostyle:ifacenames // sealed marker interface, not a behavioral -er interface
	ItemID() string
}

// AgentMessageItem is the agent's text response.
type AgentMessageItem struct {
	ID   string
	Text string
}

func (i *AgentMessageItem) ItemID() string { return i.ID }
func (*AgentMessageItem) itemType() string { return "agentMessage" }

// PlanItem is the agent's plan text.
type PlanItem struct {
	ID   string
	Text string
}

func (i *PlanItem) ItemID() string { return i.ID }
func (*PlanItem) itemType() string { return "plan" }

// ReasoningItem is the agent's reasoning summary.
type ReasoningItem struct {
	ID      string
	Summary []string
	Content []string
}

func (i *ReasoningItem) ItemID() string { return i.ID }
func (*ReasoningItem) itemType() string { return "reasoning" }

// CommandExecutionStatus represents the status of a command execution.
type CommandExecutionStatus string

const (
	CommandStatusInProgress CommandExecutionStatus = "inProgress"
	CommandStatusCompleted  CommandExecutionStatus = "completed"
	CommandStatusFailed     CommandExecutionStatus = "failed"
	CommandStatusDeclined   CommandExecutionStatus = "declined"
)

// CommandExecutionItem is a command executed by the agent.
type CommandExecutionItem struct {
	ID               string
	Command          string
	CWD              string
	Status           CommandExecutionStatus
	AggregatedOutput string
	ExitCode         *int
	DurationMs       *int
}

func (i *CommandExecutionItem) ItemID() string { return i.ID }
func (*CommandExecutionItem) itemType() string { return "commandExecution" }

// FileUpdateChange describes a single file change within a patch.
type FileUpdateChange struct {
	Path string
	Kind string // "add", "delete", "update"
}

// PatchApplyStatus is the status of a file change.
type PatchApplyStatus string

const (
	PatchStatusInProgress PatchApplyStatus = "inProgress"
	PatchStatusCompleted  PatchApplyStatus = "completed"
	PatchStatusFailed     PatchApplyStatus = "failed"
)

// FileChangeItem is a set of file changes by the agent.
type FileChangeItem struct {
	ID      string
	Changes []FileUpdateChange
	Status  PatchApplyStatus
}

func (i *FileChangeItem) ItemID() string { return i.ID }
func (*FileChangeItem) itemType() string { return "fileChange" }

// McpToolCallStatus is the status of an MCP tool call.
type McpToolCallStatus string

const (
	McpCallStatusInProgress McpToolCallStatus = "inProgress"
	McpCallStatusCompleted  McpToolCallStatus = "completed"
	McpCallStatusFailed     McpToolCallStatus = "failed"
)

// McpToolCallItem represents a call to an MCP tool.
type McpToolCallItem struct {
	ID         string
	Server     string
	Tool       string
	Arguments  any
	Result     any
	Error      *McpToolCallError
	Status     McpToolCallStatus
	DurationMs *int
}

// McpToolCallError describes an MCP tool call failure.
type McpToolCallError struct {
	Message string
}

func (i *McpToolCallItem) ItemID() string { return i.ID }
func (*McpToolCallItem) itemType() string { return "mcpToolCall" }

// DynamicToolCallStatus is the status of a dynamic tool call.
type DynamicToolCallStatus string

const (
	DynCallStatusInProgress DynamicToolCallStatus = "inProgress"
	DynCallStatusCompleted  DynamicToolCallStatus = "completed"
	DynCallStatusFailed     DynamicToolCallStatus = "failed"
)

// DynamicToolCallItem represents a dynamic tool call on the client.
type DynamicToolCallItem struct {
	ID           string
	Tool         string
	Arguments    any
	Status       DynamicToolCallStatus
	ContentItems []map[string]any
	Success      *bool
	DurationMs   *int
}

func (i *DynamicToolCallItem) ItemID() string { return i.ID }
func (*DynamicToolCallItem) itemType() string { return "dynamicToolCall" }

// WebSearchItem captures a web search request.
type WebSearchItem struct {
	ID    string
	Query string
}

func (i *WebSearchItem) ItemID() string { return i.ID }
func (*WebSearchItem) itemType() string { return "webSearch" }

// UserMessageItem represents a user message in the thread.
type UserMessageItem struct {
	ID      string
	Content []UserInput
}

func (i *UserMessageItem) ItemID() string { return i.ID }
func (*UserMessageItem) itemType() string { return "userMessage" }

// ErrorItem describes a non-fatal error surfaced as an item.
type ErrorItem struct {
	ID      string
	Message string
}

func (i *ErrorItem) ItemID() string { return i.ID }
func (*ErrorItem) itemType() string { return "error" }

// Thread represents a conversation thread.
type Thread struct {
	ID            string
	Status        string
	CWD           string
	Model         string
	ModelProvider string
	CreatedAt     float64
	UpdatedAt     float64
	Ephemeral     bool
	Turns         []Turn
}

// TurnStatus represents the status of a turn.
type TurnStatus string

const (
	TurnStatusInProgress  TurnStatus = "inProgress"
	TurnStatusCompleted   TurnStatus = "completed"
	TurnStatusFailed      TurnStatus = "failed"
	TurnStatusInterrupted TurnStatus = "interrupted"
)

// TurnErrorInfo describes a turn failure.
type TurnErrorInfo struct {
	Message string
	Details string
}

// Turn represents a single conversation turn.
type Turn struct {
	ID     string
	Items  []ThreadItem
	Status TurnStatus
	Error  *TurnErrorInfo
}

// UserInput represents input to the agent.
type UserInput struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	Path string `json:"path,omitempty"`
}

// TextInput creates a text UserInput.
func TextInput(text string) UserInput {
	return UserInput{Type: "text", Text: text}
}

// ImageInput creates a local image UserInput.
func ImageInput(path string) UserInput {
	return UserInput{Type: "localImage", Path: path}
}
