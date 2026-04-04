package codex

import (
	"context"
	"testing"
)

func TestClientStartThreadAndTurn(t *testing.T) {
	mt := newMockTransport(
		// Response to initialize (id=1)
		map[string]any{"id": 1.0, "result": map[string]any{
			"userAgent":      "codex-app-server/0.1",
			"codexHome":      "/home/.codex",
			"platformFamily": "unix",
			"platformOs":     "macos",
		}},
		// Response to thread/start (id=2)
		map[string]any{"id": 2.0, "result": map[string]any{
			"thread": map[string]any{
				"id":     "t-1",
				"status": "active",
				"cwd":    "/tmp",
				"turns":  []any{},
			},
			"model":         "o3",
			"modelProvider": "openai",
		}},
		// Response to turn/start (id=3)
		map[string]any{"id": 3.0, "result": map[string]any{
			"turn": map[string]any{
				"id":     "turn-1",
				"status": "inProgress",
				"items":  []any{},
			},
		}},
		// Notification: item/completed with agent message
		map[string]any{
			"method": "item/completed",
			"params": map[string]any{
				"threadId": "t-1",
				"turnId":   "turn-1",
				"item": map[string]any{
					"type": "agentMessage",
					"id":   "msg-1",
					"text": "Hello from Codex!",
				},
			},
		},
		// Notification: turn/completed
		map[string]any{
			"method": "turn/completed",
			"params": map[string]any{
				"threadId": "t-1",
				"turn": map[string]any{
					"id":     "turn-1",
					"status": "completed",
					"items":  []any{},
				},
			},
		},
	)

	client := &Client{
		options:   &Options{},
		transport: mt,
	}

	ctx := context.Background()

	// Manually set up session (bypass Connect which spawns subprocess).
	client.sess = newSession(ctx, mt, client.options)
	client.sess.start()

	// Initialize.
	_, err := client.sess.sendRequest(ctx, "initialize", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Start thread.
	thread, err := client.StartThread(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if thread.ID != "t-1" {
		t.Errorf("thread.ID = %q, want t-1", thread.ID)
	}
	if thread.Model != "o3" {
		t.Errorf("thread.Model = %q, want o3", thread.Model)
	}

	// Start turn and collect events.
	var events []Event
	for evt, err := range client.StartTurn(ctx, thread.ID, []UserInput{TextInput("Hello")}) {
		if err != nil {
			t.Fatal(err)
		}
		events = append(events, evt)
	}

	if len(events) != 2 {
		t.Fatalf("len(events) = %d, want 2", len(events))
	}

	// First event: item/completed with agent message.
	itemEvt, ok := events[0].(*ItemCompletedEvent)
	if !ok {
		t.Fatalf("events[0] type = %T, want *ItemCompletedEvent", events[0])
	}
	msg, ok := itemEvt.Item.(*AgentMessageItem)
	if !ok {
		t.Fatalf("item type = %T, want *AgentMessageItem", itemEvt.Item)
	}
	if msg.Text != "Hello from Codex!" {
		t.Errorf("msg.Text = %q, want 'Hello from Codex!'", msg.Text)
	}

	// Second event: turn/completed.
	_, ok = events[1].(*TurnCompletedEvent)
	if !ok {
		t.Fatalf("events[1] type = %T, want *TurnCompletedEvent", events[1])
	}

	client.Close()
}
