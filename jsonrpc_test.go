package codex

import (
	"encoding/json"
	"testing"
)

func TestClassifyMessage(t *testing.T) {
	tests := []struct {
		name string
		raw  map[string]any
		want rpcMessageKind
	}{
		{
			name: "request with id and method",
			raw:  map[string]any{"id": 1.0, "method": "thread/start", "params": map[string]any{}},
			want: rpcMessageRequest,
		},
		{
			name: "response with id and result",
			raw:  map[string]any{"id": 1.0, "result": map[string]any{"thread": map[string]any{}}},
			want: rpcMessageResponse,
		},
		{
			name: "error response with id and error",
			raw:  map[string]any{"id": 1.0, "error": map[string]any{"code": -32600.0, "message": "invalid"}},
			want: rpcMessageResponse,
		},
		{
			name: "notification with method only",
			raw:  map[string]any{"method": "turn/started", "params": map[string]any{}},
			want: rpcMessageNotification,
		},
		{
			name: "unknown message",
			raw:  map[string]any{"foo": "bar"},
			want: rpcMessageUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyMessage(tt.raw)
			if got != tt.want {
				t.Errorf("classifyMessage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRPCIDToInt(t *testing.T) {
	tests := []struct {
		name string
		v    any
		want int
	}{
		{"float64", 42.0, 42},
		{"int", 42, 42},
		{"json.Number", json.Number("42"), 42},
		{"string fallback", "abc", 0},
		{"nil", nil, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rpcIDToInt(tt.v)
			if got != tt.want {
				t.Errorf("rpcIDToInt(%v) = %v, want %v", tt.v, got, tt.want)
			}
		})
	}
}

func TestMarshalRPCRequest(t *testing.T) {
	b, err := marshalRPCRequest(1, "thread/start", map[string]any{"model": "o3"})
	if err != nil {
		t.Fatal(err)
	}
	var msg map[string]any
	if err := json.Unmarshal(b, &msg); err != nil {
		t.Fatal(err)
	}
	if msg["method"] != "thread/start" {
		t.Errorf("method = %v, want thread/start", msg["method"])
	}
	if rpcIDToInt(msg["id"]) != 1 {
		t.Errorf("id = %v, want 1", msg["id"])
	}
	params, ok := msg["params"].(map[string]any)
	if !ok {
		t.Fatal("params not a map")
	}
	if params["model"] != "o3" {
		t.Errorf("params.model = %v, want o3", params["model"])
	}
	// Should end with newline.
	if b[len(b)-1] != '\n' {
		t.Error("expected trailing newline")
	}
}

func TestMarshalRPCResponse(t *testing.T) {
	b, err := marshalRPCResponse(1, map[string]any{"ok": true})
	if err != nil {
		t.Fatal(err)
	}
	var msg map[string]any
	if err := json.Unmarshal(b, &msg); err != nil {
		t.Fatal(err)
	}
	if rpcIDToInt(msg["id"]) != 1 {
		t.Errorf("id = %v, want 1", msg["id"])
	}
	result, ok := msg["result"].(map[string]any)
	if !ok {
		t.Fatal("result not a map")
	}
	if result["ok"] != true {
		t.Errorf("result.ok = %v, want true", result["ok"])
	}
}

func TestMarshalRPCError(t *testing.T) {
	b, err := marshalRPCError(2, -32600, "invalid request")
	if err != nil {
		t.Fatal(err)
	}
	var msg map[string]any
	if err := json.Unmarshal(b, &msg); err != nil {
		t.Fatal(err)
	}
	errObj, ok := msg["error"].(map[string]any)
	if !ok {
		t.Fatal("error not a map")
	}
	if errObj["message"] != "invalid request" {
		t.Errorf("error.message = %v, want 'invalid request'", errObj["message"])
	}
}

func TestMarshalRPCRequestNilParams(t *testing.T) {
	b, err := marshalRPCRequest(1, "initialize", nil)
	if err != nil {
		t.Fatal(err)
	}
	var msg map[string]any
	if err := json.Unmarshal(b, &msg); err != nil {
		t.Fatal(err)
	}
	if _, ok := msg["params"]; ok {
		t.Error("expected no params key when nil")
	}
}
