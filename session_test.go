package codex

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
)

// mockTransport is a test transport that returns pre-loaded messages.
// Each message is delivered via an internal channel so ReadMessage blocks
// naturally between messages, allowing sendRequest to register its pending
// channel before the response arrives.
type mockTransport struct {
	mu        sync.Mutex
	writes    [][]byte
	writeCh   chan struct{}       // signals when a write occurs
	msgCh     chan map[string]any // delivers messages to ReadMessage
	closeCh   chan struct{}       // closed on Close()
	closeOnce sync.Once
}

func newMockTransport(messages ...map[string]any) *mockTransport {
	mt := &mockTransport{
		writeCh: make(chan struct{}, 16),
		msgCh:   make(chan map[string]any, len(messages)),
		closeCh: make(chan struct{}),
	}
	for _, msg := range messages {
		mt.msgCh <- msg
	}
	return mt
}

func (m *mockTransport) Connect(_ context.Context) error { return nil }

func (m *mockTransport) Write(data []byte) error {
	m.mu.Lock()
	m.writes = append(m.writes, append([]byte(nil), data...))
	m.mu.Unlock()
	select {
	case m.writeCh <- struct{}{}:
	default:
	}
	return nil
}

func (m *mockTransport) ReadMessage() (map[string]any, error) {
	select {
	case msg, ok := <-m.msgCh:
		if !ok {
			return nil, io.EOF
		}
		return msg, nil
	case <-m.closeCh:
		return nil, io.EOF
	}
}

func (m *mockTransport) Close() error {
	m.closeOnce.Do(func() { close(m.closeCh) })
	return nil
}

func TestSessionSendRequest(t *testing.T) {
	mt := newMockTransport(
		// Response to request id=1
		map[string]any{"id": 1.0, "result": map[string]any{"thread": map[string]any{"id": "t-1"}}},
	)

	ctx := context.Background()
	opts := &Options{}
	sess := newSession(ctx, mt, opts)
	sess.start()
	defer sess.close()

	result, err := sess.sendRequest(ctx, "thread/start", map[string]any{"model": "o3"})
	if err != nil {
		t.Fatal(err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	threadMap, ok := resultMap["thread"].(map[string]any)
	if !ok {
		t.Fatalf("expected thread map, got %T", resultMap["thread"])
	}
	if threadMap["id"] != "t-1" {
		t.Errorf("thread.id = %v, want t-1", threadMap["id"])
	}
}

func TestSessionNotification(t *testing.T) {
	mt := newMockTransport(
		map[string]any{
			"method": "item/agentMessage/delta",
			"params": map[string]any{
				"threadId": "t-1",
				"turnId":   "turn-1",
				"itemId":   "msg-1",
				"delta":    "hello",
			},
		},
	)

	ctx := context.Background()
	opts := &Options{}
	sess := newSession(ctx, mt, opts)
	sess.start()
	defer sess.close()

	evt, ok := <-sess.eventCh
	if !ok {
		t.Fatal("eventCh closed unexpectedly")
	}

	delta, ok := evt.(*AgentMessageDeltaEvent)
	if !ok {
		t.Fatalf("expected *AgentMessageDeltaEvent, got %T", evt)
	}
	if delta.Delta != "hello" {
		t.Errorf("Delta = %q, want hello", delta.Delta)
	}
}

func TestSessionServerRequestCommandApproval(t *testing.T) {
	done := make(chan struct{})
	mt := newMockTransport(
		// Server request: command approval
		map[string]any{
			"id":     100.0,
			"method": "item/commandExecution/requestApproval",
			"params": map[string]any{
				"threadId": "t-1",
				"turnId":   "turn-1",
				"itemId":   "cmd-1",
				"command":  "rm -rf /",
				"cwd":      "/tmp",
			},
		},
	)

	ctx := context.Background()
	opts := &Options{
		OnCommandApproval: func(_ context.Context, req CommandApprovalRequest) (ApprovalDecision, error) {
			defer close(done)
			if req.Command == "rm -rf /" {
				return DecisionDecline, nil
			}
			return DecisionAccept, nil
		},
	}
	sess := newSession(ctx, mt, opts)
	sess.start()
	defer sess.close()

	// Wait for the callback to complete and the response to be written.
	<-done
	<-mt.writeCh

	// Check that a response was written.
	mt.mu.Lock()
	defer mt.mu.Unlock()
	if len(mt.writes) == 0 {
		t.Fatal("expected a response to be written")
	}
	// The response should contain "decline".
	response := string(mt.writes[0])
	if !contains(response, "decline") {
		t.Errorf("response = %s, want to contain 'decline'", response)
	}
}

func TestSessionServerRequestDefaultDecline(t *testing.T) {
	mt := newMockTransport(
		map[string]any{
			"id":     1.0,
			"method": "item/commandExecution/requestApproval",
			"params": map[string]any{
				"threadId": "t-1",
				"turnId":   "turn-1",
				"itemId":   "cmd-1",
				"command":  "ls",
			},
		},
	)

	ctx := context.Background()
	// No callback registered → should default to decline.
	opts := &Options{}
	sess := newSession(ctx, mt, opts)
	sess.start()
	defer sess.close()

	// Wait for the server request handler goroutine to write the response.
	<-mt.writeCh

	mt.mu.Lock()
	defer mt.mu.Unlock()
	if len(mt.writes) == 0 {
		t.Fatal("expected a response")
	}
	if !contains(string(mt.writes[0]), "decline") {
		t.Errorf("expected decline, got %s", mt.writes[0])
	}
}

func TestSessionRPCError(t *testing.T) {
	mt := newMockTransport(
		map[string]any{
			"id": 1.0,
			"error": map[string]any{
				"code":    -32600.0,
				"message": "invalid request",
			},
		},
	)

	ctx := context.Background()
	sess := newSession(ctx, mt, &Options{})
	sess.start()
	defer sess.close()

	_, err := sess.sendRequest(ctx, "bad/method", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	var rpcErr *RPCError
	if !errors.As(err, &rpcErr) {
		t.Fatalf("expected *RPCError, got %T", err)
	}
	if rpcErr.Code != -32600 {
		t.Errorf("Code = %d, want -32600", rpcErr.Code)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
