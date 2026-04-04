package codex

import "testing"

func TestParseEventThreadStarted(t *testing.T) {
	params := map[string]any{
		"threadId": "t-1",
		"thread": map[string]any{
			"id":     "t-1",
			"status": "active",
			"cwd":    "/tmp",
		},
	}
	evt, err := parseEvent("thread/started", params)
	if err != nil {
		t.Fatal(err)
	}
	e, ok := evt.(*ThreadStartedEvent)
	if !ok {
		t.Fatalf("expected *ThreadStartedEvent, got %T", evt)
	}
	if e.ThreadID != "t-1" {
		t.Errorf("ThreadID = %q, want t-1", e.ThreadID)
	}
	if e.Thread == nil || e.Thread.ID != "t-1" {
		t.Errorf("Thread.ID = %v, want t-1", e.Thread)
	}
}

func TestParseEventTurnCompleted(t *testing.T) {
	params := map[string]any{
		"threadId": "t-1",
		"turn": map[string]any{
			"id":     "turn-1",
			"status": "completed",
			"items":  []any{},
		},
	}
	evt, err := parseEvent("turn/completed", params)
	if err != nil {
		t.Fatal(err)
	}
	e, ok := evt.(*TurnCompletedEvent)
	if !ok {
		t.Fatalf("expected *TurnCompletedEvent, got %T", evt)
	}
	if e.Turn == nil || e.Turn.Status != TurnStatusCompleted {
		t.Errorf("Turn.Status = %v, want completed", e.Turn)
	}
}

func TestParseEventItemCompleted(t *testing.T) {
	params := map[string]any{
		"threadId": "t-1",
		"turnId":   "turn-1",
		"item": map[string]any{
			"type": "agentMessage",
			"id":   "msg-1",
			"text": "Hello!",
		},
	}
	evt, err := parseEvent("item/completed", params)
	if err != nil {
		t.Fatal(err)
	}
	e, ok := evt.(*ItemCompletedEvent)
	if !ok {
		t.Fatalf("expected *ItemCompletedEvent, got %T", evt)
	}
	msg, ok := e.Item.(*AgentMessageItem)
	if !ok {
		t.Fatalf("expected *AgentMessageItem, got %T", e.Item)
	}
	if msg.Text != "Hello!" {
		t.Errorf("Text = %q, want Hello!", msg.Text)
	}
}

func TestParseEventAgentMessageDelta(t *testing.T) {
	params := map[string]any{
		"threadId": "t-1",
		"turnId":   "turn-1",
		"itemId":   "msg-1",
		"delta":    "chunk",
	}
	evt, err := parseEvent("item/agentMessage/delta", params)
	if err != nil {
		t.Fatal(err)
	}
	e, ok := evt.(*AgentMessageDeltaEvent)
	if !ok {
		t.Fatalf("expected *AgentMessageDeltaEvent, got %T", evt)
	}
	if e.Delta != "chunk" {
		t.Errorf("Delta = %q, want chunk", e.Delta)
	}
}

func TestParseEventUnknown(t *testing.T) {
	evt, err := parseEvent("some/unknown/event", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if evt != nil {
		t.Errorf("expected nil for unknown event, got %T", evt)
	}
}

func TestParseThreadItemCommandExecution(t *testing.T) {
	exitCode := 0.0
	raw := map[string]any{
		"type":             "commandExecution",
		"id":               "cmd-1",
		"command":          "ls -la",
		"cwd":              "/tmp",
		"status":           "completed",
		"aggregatedOutput": "total 0\n",
		"exitCode":         exitCode,
		"durationMs":       150.0,
	}
	item, err := parseThreadItem(raw)
	if err != nil {
		t.Fatal(err)
	}
	cmd, ok := item.(*CommandExecutionItem)
	if !ok {
		t.Fatalf("expected *CommandExecutionItem, got %T", item)
	}
	if cmd.Command != "ls -la" {
		t.Errorf("Command = %q, want ls -la", cmd.Command)
	}
	if cmd.ExitCode == nil || *cmd.ExitCode != 0 {
		t.Errorf("ExitCode = %v, want 0", cmd.ExitCode)
	}
	if cmd.DurationMs == nil || *cmd.DurationMs != 150 {
		t.Errorf("DurationMs = %v, want 150", cmd.DurationMs)
	}
}

func TestParseThreadItemFileChange(t *testing.T) {
	raw := map[string]any{
		"type":   "fileChange",
		"id":     "fc-1",
		"status": "completed",
		"changes": []any{
			map[string]any{"path": "main.go", "kind": "update"},
			map[string]any{"path": "new.go", "kind": "add"},
		},
	}
	item, err := parseThreadItem(raw)
	if err != nil {
		t.Fatal(err)
	}
	fc, ok := item.(*FileChangeItem)
	if !ok {
		t.Fatalf("expected *FileChangeItem, got %T", item)
	}
	if len(fc.Changes) != 2 {
		t.Fatalf("len(Changes) = %d, want 2", len(fc.Changes))
	}
	if fc.Changes[0].Path != "main.go" || fc.Changes[0].Kind != "update" {
		t.Errorf("Changes[0] = %+v", fc.Changes[0])
	}
}

func TestParseThreadItemMcpToolCall(t *testing.T) {
	raw := map[string]any{
		"type":      "mcpToolCall",
		"id":        "mcp-1",
		"server":    "my-server",
		"tool":      "search",
		"arguments": map[string]any{"q": "test"},
		"status":    "completed",
		"error":     map[string]any{"message": "timeout"},
	}
	item, err := parseThreadItem(raw)
	if err != nil {
		t.Fatal(err)
	}
	mcp, ok := item.(*McpToolCallItem)
	if !ok {
		t.Fatalf("expected *McpToolCallItem, got %T", item)
	}
	if mcp.Server != "my-server" {
		t.Errorf("Server = %q, want my-server", mcp.Server)
	}
	if mcp.Error == nil || mcp.Error.Message != "timeout" {
		t.Errorf("Error = %+v, want timeout", mcp.Error)
	}
}

func TestParseThreadItemWebSearch(t *testing.T) {
	raw := map[string]any{
		"type":  "webSearch",
		"id":    "ws-1",
		"query": "golang concurrency",
	}
	item, err := parseThreadItem(raw)
	if err != nil {
		t.Fatal(err)
	}
	ws, ok := item.(*WebSearchItem)
	if !ok {
		t.Fatalf("expected *WebSearchItem, got %T", item)
	}
	if ws.Query != "golang concurrency" {
		t.Errorf("Query = %q, want golang concurrency", ws.Query)
	}
}

func TestParseThreadItemUnknown(t *testing.T) {
	raw := map[string]any{
		"type": "futureType",
		"id":   "x-1",
	}
	item, err := parseThreadItem(raw)
	if err != nil {
		t.Fatal(err)
	}
	if item != nil {
		t.Errorf("expected nil for unknown item type, got %T", item)
	}
}

func TestParseTurnWithError(t *testing.T) {
	raw := map[string]any{
		"id":     "turn-1",
		"status": "failed",
		"error": map[string]any{
			"message":           "rate limit exceeded",
			"additionalDetails": "retry after 30s",
		},
		"items": []any{},
	}
	turn := parseTurn(raw)
	if turn.Status != TurnStatusFailed {
		t.Errorf("Status = %v, want failed", turn.Status)
	}
	if turn.Error == nil {
		t.Fatal("expected Error to be non-nil")
	}
	if turn.Error.Message != "rate limit exceeded" {
		t.Errorf("Error.Message = %q", turn.Error.Message)
	}
}
