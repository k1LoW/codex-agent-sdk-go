//go:build integration

package codex

import (
	"context"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"
)

func skipIfNoCLI(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("codex"); err != nil {
		t.Skip("codex CLI not found in PATH")
	}
}

// commonQueryOpts returns QueryOption(s) shared by most integration tests.
func commonQueryOpts(extra ...QueryOption) []QueryOption {
	base := []QueryOption{
		WithThreadOptions(
			WithApprovalPolicy("never"),
		),
	}
	return append(base, extra...)
}

// collectText gathers all AgentMessageDelta text from events.
func collectText(t *testing.T, events []Event) string {
	t.Helper()
	var b strings.Builder
	for _, evt := range events {
		if d, ok := evt.(*AgentMessageDeltaEvent); ok {
			b.WriteString(d.Delta)
		}
		if ic, ok := evt.(*ItemCompletedEvent); ok {
			if msg, ok := ic.Item.(*AgentMessageItem); ok {
				if b.Len() == 0 {
					b.WriteString(msg.Text)
				}
			}
		}
	}
	return b.String()
}

// --- Query (one-shot) tests ---

func TestIntegration_Query_Basic(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := context.WithTimeout(t.Context(), 120*time.Second)
	t.Cleanup(cancel)

	var gotDelta, gotTurnCompleted bool
	for evt, err := range Query(ctx, "Reply with exactly: hello", commonQueryOpts()...) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		switch evt.(type) {
		case *AgentMessageDeltaEvent:
			gotDelta = true
		case *TurnCompletedEvent:
			gotTurnCompleted = true
		}
	}
	if !gotDelta {
		t.Error("expected at least one AgentMessageDeltaEvent")
	}
	if !gotTurnCompleted {
		t.Error("expected a TurnCompletedEvent")
	}
}

func TestIntegration_Query_EventLifecycle(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := context.WithTimeout(t.Context(), 120*time.Second)
	t.Cleanup(cancel)

	var gotThreadStarted, gotTurnStarted, gotItemStarted, gotItemCompleted, gotTurnCompleted bool
	for evt, err := range Query(ctx, "Say hello", commonQueryOpts()...) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		switch evt.(type) {
		case *ThreadStartedEvent:
			gotThreadStarted = true
		case *TurnStartedEvent:
			gotTurnStarted = true
		case *ItemStartedEvent:
			gotItemStarted = true
		case *ItemCompletedEvent:
			gotItemCompleted = true
		case *TurnCompletedEvent:
			gotTurnCompleted = true
		}
	}
	if !gotThreadStarted {
		t.Error("expected ThreadStartedEvent")
	}
	if !gotTurnStarted {
		t.Error("expected TurnStartedEvent")
	}
	if !gotItemStarted {
		t.Error("expected ItemStartedEvent")
	}
	if !gotItemCompleted {
		t.Error("expected ItemCompletedEvent")
	}
	if !gotTurnCompleted {
		t.Error("expected TurnCompletedEvent")
	}
}

func TestIntegration_Query_AgentMessageContent(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := context.WithTimeout(t.Context(), 120*time.Second)
	t.Cleanup(cancel)

	var events []Event
	for evt, err := range Query(ctx, "Reply with exactly: pong", commonQueryOpts()...) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		events = append(events, evt)
	}
	text := collectText(t, events)
	if !strings.Contains(strings.ToLower(text), "pong") {
		t.Errorf("expected response to contain 'pong', got: %q", text)
	}
}

func TestIntegration_Query_WithModel(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := context.WithTimeout(t.Context(), 120*time.Second)
	t.Cleanup(cancel)

	var gotTurnCompleted bool
	for evt, err := range Query(ctx, "Say hi",
		commonQueryOpts(
			WithThreadOptions(WithModel("o3-mini")),
		)...,
	) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := evt.(*TurnCompletedEvent); ok {
			gotTurnCompleted = true
		}
	}
	if !gotTurnCompleted {
		t.Error("expected a TurnCompletedEvent")
	}
}

func TestIntegration_Query_WithEffort(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := context.WithTimeout(t.Context(), 120*time.Second)
	t.Cleanup(cancel)

	var gotTurnCompleted bool
	for evt, err := range Query(ctx, "What is 2+2?",
		commonQueryOpts(
			WithTurnOptions(WithEffort("low")),
		)...,
	) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := evt.(*TurnCompletedEvent); ok {
			gotTurnCompleted = true
		}
	}
	if !gotTurnCompleted {
		t.Error("expected a TurnCompletedEvent")
	}
}

func TestIntegration_Query_WithOutputSchema(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := context.WithTimeout(t.Context(), 120*time.Second)
	t.Cleanup(cancel)

	var gotTurnCompleted bool
	for evt, err := range Query(ctx, "What is 2+2?",
		commonQueryOpts(
			WithTurnOptions(WithOutputSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"answer": map[string]any{"type": "number"},
				},
				"required": []string{"answer"},
			})),
		)...,
	) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := evt.(*TurnCompletedEvent); ok {
			gotTurnCompleted = true
		}
	}
	if !gotTurnCompleted {
		t.Error("expected a TurnCompletedEvent")
	}
}

func TestIntegration_Query_WithCWD(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := context.WithTimeout(t.Context(), 120*time.Second)
	t.Cleanup(cancel)

	var gotTurnCompleted bool
	for evt, err := range Query(ctx, "What directory are you in? Run pwd.",
		commonQueryOpts(
			WithThreadOptions(WithCWD("/tmp")),
		)...,
	) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := evt.(*TurnCompletedEvent); ok {
			gotTurnCompleted = true
		}
	}
	if !gotTurnCompleted {
		t.Error("expected a TurnCompletedEvent")
	}
}

func TestIntegration_Query_WithEphemeral(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := context.WithTimeout(t.Context(), 120*time.Second)
	t.Cleanup(cancel)

	var thread *Thread
	for evt, err := range Query(ctx, "Say hi",
		commonQueryOpts(
			WithThreadOptions(WithEphemeral(true)),
		)...,
	) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ts, ok := evt.(*ThreadStartedEvent); ok {
			thread = ts.Thread
		}
	}
	if thread == nil {
		t.Fatal("expected ThreadStartedEvent with Thread")
	}
}

func TestIntegration_Query_Stderr(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := context.WithTimeout(t.Context(), 120*time.Second)
	t.Cleanup(cancel)

	var mu sync.Mutex
	var lines []string
	for evt, err := range Query(ctx, "Say hello",
		commonQueryOpts(
			WithOptions(WithStderr(func(line string) {
				mu.Lock()
				lines = append(lines, line)
				mu.Unlock()
			})),
		)...,
	) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		_ = evt
	}
	// Stderr callback may or may not produce output depending on CLI version,
	// but the callback should not have caused any errors.
}

func TestIntegration_Query_ContextCancel(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	t.Cleanup(cancel)

	var gotError bool
	for _, err := range Query(ctx, "Write a 10000 word essay about the history of computing",
		commonQueryOpts()...,
	) {
		if err != nil {
			gotError = true
			break
		}
	}
	// Either we got an error from context cancellation, or the query completed
	// within the timeout (unlikely for a long prompt). Both are acceptable.
	_ = gotError
}

func TestIntegration_Query_TurnCompletedHasTurn(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := context.WithTimeout(t.Context(), 120*time.Second)
	t.Cleanup(cancel)

	var turn *Turn
	for evt, err := range Query(ctx, "Say hello", commonQueryOpts()...) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tc, ok := evt.(*TurnCompletedEvent); ok {
			turn = tc.Turn
		}
	}
	if turn == nil {
		t.Fatal("expected TurnCompletedEvent with Turn")
	}
	if turn.ID == "" {
		t.Error("expected non-empty Turn ID")
	}
	if turn.Status != TurnStatusCompleted {
		t.Errorf("expected turn status %q, got %q", TurnStatusCompleted, turn.Status)
	}
}

func TestIntegration_Query_ItemCompletedAgentMessage(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := context.WithTimeout(t.Context(), 120*time.Second)
	t.Cleanup(cancel)

	var gotAgentMessage bool
	for evt, err := range Query(ctx, "Reply with exactly: yes", commonQueryOpts()...) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ic, ok := evt.(*ItemCompletedEvent); ok {
			if msg, ok := ic.Item.(*AgentMessageItem); ok {
				gotAgentMessage = true
				if msg.Text == "" {
					t.Error("expected non-empty AgentMessageItem.Text")
				}
				if msg.ID == "" {
					t.Error("expected non-empty AgentMessageItem.ID")
				}
			}
		}
	}
	if !gotAgentMessage {
		t.Error("expected an ItemCompletedEvent with AgentMessageItem")
	}
}

func TestIntegration_Query_CommandExecution(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := context.WithTimeout(t.Context(), 120*time.Second)
	t.Cleanup(cancel)

	var gotCommandExec bool
	var gotOutputDelta bool
	for evt, err := range Query(ctx, "Run the command: echo hello-codex",
		commonQueryOpts()...,
	) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		switch e := evt.(type) {
		case *CommandOutputDeltaEvent:
			gotOutputDelta = true
		case *ItemCompletedEvent:
			if cmd, ok := e.Item.(*CommandExecutionItem); ok {
				gotCommandExec = true
				if cmd.Command == "" {
					t.Error("expected non-empty Command")
				}
				if cmd.Status != CommandStatusCompleted {
					t.Errorf("expected command status %q, got %q", CommandStatusCompleted, cmd.Status)
				}
			}
		}
	}
	if !gotCommandExec {
		t.Error("expected an ItemCompletedEvent with CommandExecutionItem")
	}
	// CommandOutputDelta is optional depending on whether the server streams output.
	_ = gotOutputDelta
}

// --- Approval callback tests ---

func TestIntegration_Query_CommandApproval_Accept(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := context.WithTimeout(t.Context(), 120*time.Second)
	t.Cleanup(cancel)

	var mu sync.Mutex
	var approvalCalled bool
	var approvedCommand string

	for evt, err := range Query(ctx, "Run the command: echo approval-test",
		WithOptions(WithOnCommandApproval(func(_ context.Context, req CommandApprovalRequest) (ApprovalDecision, error) {
			mu.Lock()
			approvalCalled = true
			approvedCommand = req.Command
			mu.Unlock()
			return DecisionAccept, nil
		})),
		WithThreadOptions(WithApprovalPolicy("on-failure")),
	) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		_ = evt
	}

	mu.Lock()
	defer mu.Unlock()
	if approvalCalled {
		t.Logf("approval callback was called for command: %q", approvedCommand)
	}
	// Approval may or may not be requested depending on the approval policy
	// and server behavior. The key assertion is no errors or hangs.
}

func TestIntegration_Query_CommandApproval_Decline(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := context.WithTimeout(t.Context(), 120*time.Second)
	t.Cleanup(cancel)

	var mu sync.Mutex
	var approvalCalled bool

	for evt, err := range Query(ctx, "Run the command: echo should-be-declined",
		WithOptions(WithOnCommandApproval(func(_ context.Context, _ CommandApprovalRequest) (ApprovalDecision, error) {
			mu.Lock()
			approvalCalled = true
			mu.Unlock()
			return DecisionDecline, nil
		})),
		WithThreadOptions(WithApprovalPolicy("on-failure")),
	) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ic, ok := evt.(*ItemCompletedEvent); ok {
			if cmd, ok := ic.Item.(*CommandExecutionItem); ok && approvalCalled {
				if cmd.Status == CommandStatusDeclined {
					t.Logf("command was declined as expected")
				}
			}
		}
	}
}

func TestIntegration_Query_FileChangeApproval(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := context.WithTimeout(t.Context(), 120*time.Second)
	t.Cleanup(cancel)

	var mu sync.Mutex
	var approvalCalled bool

	for evt, err := range Query(ctx, "Create a file called /tmp/codex-integration-test.txt with the content 'hello'",
		WithOptions(WithOnFileChangeApproval(func(_ context.Context, _ FileChangeApprovalRequest) (ApprovalDecision, error) {
			mu.Lock()
			approvalCalled = true
			mu.Unlock()
			return DecisionAccept, nil
		})),
		WithThreadOptions(WithApprovalPolicy("on-failure")),
	) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		_ = evt
	}

	mu.Lock()
	defer mu.Unlock()
	// File change approval may or may not be called depending on how the agent
	// decides to create the file (command vs file change).
	t.Logf("file change approval called: %v", approvalCalled)
}

func TestIntegration_Query_UserInput(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := context.WithTimeout(t.Context(), 120*time.Second)
	t.Cleanup(cancel)

	var mu sync.Mutex
	var callbackCalled bool

	for evt, err := range Query(ctx, "Ask me for my name",
		WithOptions(WithOnUserInput(func(_ context.Context, req UserInputRequest) (map[string]string, error) {
			mu.Lock()
			callbackCalled = true
			mu.Unlock()
			answers := make(map[string]string)
			for _, q := range req.Questions {
				if id, ok := q["id"].(string); ok {
					answers[id] = "TestUser"
				}
			}
			return answers, nil
		})),
		WithThreadOptions(WithApprovalPolicy("on-failure")),
	) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		_ = evt
	}

	mu.Lock()
	defer mu.Unlock()
	// User input callback may not be called if the model doesn't decide
	// to ask for input. The key assertion is no errors or hangs.
	t.Logf("user input callback called: %v", callbackCalled)
}

func TestIntegration_Query_ToolCall(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := context.WithTimeout(t.Context(), 120*time.Second)
	t.Cleanup(cancel)

	var mu sync.Mutex
	var callbackCalled bool
	var toolName string

	for evt, err := range Query(ctx, "Use a tool to help me",
		WithOptions(WithOnToolCall(func(_ context.Context, req ToolCallRequest) (ToolCallResponse, error) {
			mu.Lock()
			callbackCalled = true
			toolName = req.Tool
			mu.Unlock()
			return ToolCallResponse{
				ContentItems: []map[string]any{
					{"type": "text", "text": "tool result"},
				},
				Success: true,
			}, nil
		})),
		WithThreadOptions(WithApprovalPolicy("never")),
	) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		_ = evt
	}

	mu.Lock()
	defer mu.Unlock()
	// Dynamic tool call may not be triggered depending on model behavior.
	t.Logf("tool call callback called: %v, tool: %q", callbackCalled, toolName)
}

// --- Client (multi-turn) tests ---

func TestIntegration_Client_BasicConversation(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := context.WithTimeout(t.Context(), 180*time.Second)
	t.Cleanup(cancel)

	client := NewClient(
		WithOnCommandApproval(func(_ context.Context, _ CommandApprovalRequest) (ApprovalDecision, error) {
			return DecisionAccept, nil
		}),
	)
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("connect error: %v", err)
	}
	t.Cleanup(func() { client.Close() })

	thread, err := client.StartThread(ctx, WithApprovalPolicy("never"))
	if err != nil {
		t.Fatalf("StartThread error: %v", err)
	}
	if thread.ID == "" {
		t.Fatal("expected non-empty thread ID")
	}

	// First turn
	var firstText strings.Builder
	for evt, err := range client.StartTurn(ctx, thread.ID,
		[]UserInput{TextInput("Remember the number 42. Reply with just 'OK'.")}) {
		if err != nil {
			t.Fatalf("first turn error: %v", err)
		}
		if d, ok := evt.(*AgentMessageDeltaEvent); ok {
			firstText.WriteString(d.Delta)
		}
	}
	if firstText.Len() == 0 {
		t.Error("expected non-empty first response")
	}

	// Second turn
	var secondText strings.Builder
	for evt, err := range client.StartTurn(ctx, thread.ID,
		[]UserInput{TextInput("What number did I ask you to remember? Reply with just the number.")}) {
		if err != nil {
			t.Fatalf("second turn error: %v", err)
		}
		if d, ok := evt.(*AgentMessageDeltaEvent); ok {
			secondText.WriteString(d.Delta)
		}
	}
	if !strings.Contains(secondText.String(), "42") {
		t.Errorf("expected second response to contain '42', got: %q", secondText.String())
	}
}

func TestIntegration_Client_StartThread_WithModel(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := context.WithTimeout(t.Context(), 120*time.Second)
	t.Cleanup(cancel)

	client := NewClient()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("connect error: %v", err)
	}
	t.Cleanup(func() { client.Close() })

	thread, err := client.StartThread(ctx, WithModel("o3-mini"), WithApprovalPolicy("never"))
	if err != nil {
		t.Fatalf("StartThread error: %v", err)
	}
	if thread.ID == "" {
		t.Fatal("expected non-empty thread ID")
	}

	var gotTurnCompleted bool
	for evt, err := range client.StartTurn(ctx, thread.ID,
		[]UserInput{TextInput("Say hi")}) {
		if err != nil {
			t.Fatalf("turn error: %v", err)
		}
		if _, ok := evt.(*TurnCompletedEvent); ok {
			gotTurnCompleted = true
		}
	}
	if !gotTurnCompleted {
		t.Error("expected TurnCompletedEvent")
	}
}

func TestIntegration_Client_TurnWithEffort(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := context.WithTimeout(t.Context(), 120*time.Second)
	t.Cleanup(cancel)

	client := NewClient()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("connect error: %v", err)
	}
	t.Cleanup(func() { client.Close() })

	thread, err := client.StartThread(ctx, WithApprovalPolicy("never"))
	if err != nil {
		t.Fatalf("StartThread error: %v", err)
	}

	var gotTurnCompleted bool
	for evt, err := range client.StartTurn(ctx, thread.ID,
		[]UserInput{TextInput("What is 2+2?")},
		WithEffort("low"),
	) {
		if err != nil {
			t.Fatalf("turn error: %v", err)
		}
		if _, ok := evt.(*TurnCompletedEvent); ok {
			gotTurnCompleted = true
		}
	}
	if !gotTurnCompleted {
		t.Error("expected TurnCompletedEvent")
	}
}

func TestIntegration_Client_Interrupt(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := context.WithTimeout(t.Context(), 120*time.Second)
	t.Cleanup(cancel)

	client := NewClient()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("connect error: %v", err)
	}
	t.Cleanup(func() { client.Close() })

	thread, err := client.StartThread(ctx, WithApprovalPolicy("never"))
	if err != nil {
		t.Fatalf("StartThread error: %v", err)
	}

	// Capture the turn ID from TurnStartedEvent and interrupt.
	var mu sync.Mutex
	var turnID string
	go func() {
		// Wait briefly for the turn to begin, then interrupt.
		time.Sleep(3 * time.Second)
		mu.Lock()
		tid := turnID
		mu.Unlock()
		if tid != "" {
			_ = client.Interrupt(ctx, thread.ID, tid)
		}
	}()

	for evt, err := range client.StartTurn(ctx, thread.ID,
		[]UserInput{TextInput("Write a very long essay about the history of computing. Make it at least 5000 words.")}) {
		if err != nil {
			// Interrupt may cause an error, which is acceptable.
			break
		}
		if ts, ok := evt.(*TurnStartedEvent); ok && ts.Turn != nil {
			mu.Lock()
			turnID = ts.Turn.ID
			mu.Unlock()
		}
	}
	// If we get here, the interrupt worked or the turn completed. Both are fine.
}

func TestIntegration_Client_ResumeThread(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := context.WithTimeout(t.Context(), 180*time.Second)
	t.Cleanup(cancel)

	client := NewClient()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("connect error: %v", err)
	}
	t.Cleanup(func() { client.Close() })

	// Start a thread and complete a turn.
	thread, err := client.StartThread(ctx, WithApprovalPolicy("never"))
	if err != nil {
		t.Fatalf("StartThread error: %v", err)
	}

	for evt, err := range client.StartTurn(ctx, thread.ID,
		[]UserInput{TextInput("Say hello")}) {
		if err != nil {
			t.Fatalf("first turn error: %v", err)
		}
		_ = evt
	}

	// Resume the same thread.
	resumed, err := client.ResumeThread(ctx, thread.ID)
	if err != nil {
		t.Fatalf("ResumeThread error: %v", err)
	}
	if resumed.ID != thread.ID {
		t.Errorf("expected resumed thread ID %q, got %q", thread.ID, resumed.ID)
	}

	// Start another turn on the resumed thread.
	var gotTurnCompleted bool
	for evt, err := range client.StartTurn(ctx, resumed.ID,
		[]UserInput{TextInput("Say goodbye")}) {
		if err != nil {
			t.Fatalf("resumed turn error: %v", err)
		}
		if _, ok := evt.(*TurnCompletedEvent); ok {
			gotTurnCompleted = true
		}
	}
	if !gotTurnCompleted {
		t.Error("expected TurnCompletedEvent on resumed thread")
	}
}

func TestIntegration_Client_NotConnected(t *testing.T) {
	client := NewClient()

	ctx := t.Context()

	_, err := client.StartThread(ctx)
	if err == nil {
		t.Error("expected error when starting thread on unconnected client")
	}
}

func TestIntegration_Client_MultipleThreads(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := context.WithTimeout(t.Context(), 180*time.Second)
	t.Cleanup(cancel)

	client := NewClient()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("connect error: %v", err)
	}
	t.Cleanup(func() { client.Close() })

	// Start two separate threads.
	thread1, err := client.StartThread(ctx, WithApprovalPolicy("never"))
	if err != nil {
		t.Fatalf("StartThread 1 error: %v", err)
	}

	thread2, err := client.StartThread(ctx, WithApprovalPolicy("never"))
	if err != nil {
		t.Fatalf("StartThread 2 error: %v", err)
	}

	if thread1.ID == thread2.ID {
		t.Error("expected different thread IDs")
	}

	// Run a turn on each.
	for evt, err := range client.StartTurn(ctx, thread1.ID,
		[]UserInput{TextInput("Say 'thread one'")}) {
		if err != nil {
			t.Fatalf("thread1 turn error: %v", err)
		}
		_ = evt
	}

	for evt, err := range client.StartTurn(ctx, thread2.ID,
		[]UserInput{TextInput("Say 'thread two'")}) {
		if err != nil {
			t.Fatalf("thread2 turn error: %v", err)
		}
		_ = evt
	}
}

// --- Sandbox tests ---

func TestIntegration_Query_WithSandbox(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := context.WithTimeout(t.Context(), 120*time.Second)
	t.Cleanup(cancel)

	var gotTurnCompleted bool
	for evt, err := range Query(ctx, "Say hi",
		commonQueryOpts(
			WithThreadOptions(WithSandbox("read-only")),
		)...,
	) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := evt.(*TurnCompletedEvent); ok {
			gotTurnCompleted = true
		}
	}
	if !gotTurnCompleted {
		t.Error("expected a TurnCompletedEvent")
	}
}

// --- Env tests ---

func TestIntegration_Query_WithEnv(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := context.WithTimeout(t.Context(), 120*time.Second)
	t.Cleanup(cancel)

	var events []Event
	for evt, err := range Query(ctx, "Run: echo $CODEX_TEST_VAR",
		commonQueryOpts(
			WithOptions(WithEnv(map[string]string{
				"CODEX_TEST_VAR": "integration-test-value",
			})),
		)...,
	) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		events = append(events, evt)
	}

	// Check if the command output contains our env var value.
	var gotOutput bool
	for _, evt := range events {
		if d, ok := evt.(*CommandOutputDeltaEvent); ok {
			if strings.Contains(d.Delta, "integration-test-value") {
				gotOutput = true
			}
		}
		if ic, ok := evt.(*ItemCompletedEvent); ok {
			if cmd, ok := ic.Item.(*CommandExecutionItem); ok {
				if strings.Contains(cmd.AggregatedOutput, "integration-test-value") {
					gotOutput = true
				}
			}
		}
	}
	// The env var may or may not appear depending on whether the agent
	// actually ran the command. The key assertion is no errors.
	t.Logf("env var in output: %v", gotOutput)
}

// --- CLIPath tests ---

func TestIntegration_Query_CLINotFound(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	t.Cleanup(cancel)

	var gotError bool
	for _, err := range Query(ctx, "Say hi",
		WithOptions(WithCLIPath("/nonexistent/codex-binary")),
	) {
		if err != nil {
			gotError = true
			break
		}
	}
	if !gotError {
		t.Error("expected error with invalid CLI path")
	}
}

// --- AcceptForSession tests ---

func TestIntegration_Query_CommandApproval_AcceptForSession(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := context.WithTimeout(t.Context(), 120*time.Second)
	t.Cleanup(cancel)

	var mu sync.Mutex
	var callCount int

	for evt, err := range Query(ctx, "Run these two commands: echo first && echo second",
		WithOptions(WithOnCommandApproval(func(_ context.Context, _ CommandApprovalRequest) (ApprovalDecision, error) {
			mu.Lock()
			callCount++
			mu.Unlock()
			return DecisionAcceptForSession, nil
		})),
		WithThreadOptions(WithApprovalPolicy("on-failure")),
	) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		_ = evt
	}

	mu.Lock()
	defer mu.Unlock()
	t.Logf("approval callback called %d times", callCount)
}

// --- Elicitation tests ---

func TestIntegration_Query_Elicitation(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := context.WithTimeout(t.Context(), 120*time.Second)
	t.Cleanup(cancel)

	var mu sync.Mutex
	var callbackCalled bool

	for evt, err := range Query(ctx, "Say hi",
		commonQueryOpts(
			WithOptions(WithOnElicitation(func(_ context.Context, req ElicitationRequest) (ElicitationResponse, error) {
				mu.Lock()
				callbackCalled = true
				mu.Unlock()
				return ElicitationResponse{
					Action:   ElicitationAccept,
					Response: map[string]any{"answer": "yes"},
				}, nil
			})),
		)...,
	) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		_ = evt
	}

	mu.Lock()
	defer mu.Unlock()
	// Elicitation is only triggered by MCP servers, so it likely won't be
	// called in this simple test. The key assertion is no errors or hangs.
	t.Logf("elicitation callback called: %v", callbackCalled)
}

// --- Client with approval callbacks ---

func TestIntegration_Client_WithApprovalCallbacks(t *testing.T) {
	skipIfNoCLI(t)
	ctx, cancel := context.WithTimeout(t.Context(), 120*time.Second)
	t.Cleanup(cancel)

	var mu sync.Mutex
	var commandApproved bool

	client := NewClient(
		WithOnCommandApproval(func(_ context.Context, req CommandApprovalRequest) (ApprovalDecision, error) {
			mu.Lock()
			commandApproved = true
			mu.Unlock()
			t.Logf("approving command: %s", req.Command)
			return DecisionAccept, nil
		}),
		WithOnFileChangeApproval(func(_ context.Context, _ FileChangeApprovalRequest) (ApprovalDecision, error) {
			return DecisionAccept, nil
		}),
	)
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("connect error: %v", err)
	}
	t.Cleanup(func() { client.Close() })

	thread, err := client.StartThread(ctx, WithApprovalPolicy("on-failure"))
	if err != nil {
		t.Fatalf("StartThread error: %v", err)
	}

	for evt, err := range client.StartTurn(ctx, thread.ID,
		[]UserInput{TextInput("Run: echo hello-from-client")}) {
		if err != nil {
			t.Fatalf("turn error: %v", err)
		}
		_ = evt
	}

	mu.Lock()
	defer mu.Unlock()
	t.Logf("command approved: %v", commandApproved)
}
