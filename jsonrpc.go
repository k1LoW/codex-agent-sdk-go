package codex

import "encoding/json"

// rpcMessageKind identifies the type of a JSON-RPC message.
type rpcMessageKind int

const (
	rpcMessageRequest rpcMessageKind = iota
	rpcMessageResponse
	rpcMessageNotification
	rpcMessageUnknown
)

// classifyMessage determines the kind of a raw JSON-RPC message.
func classifyMessage(raw map[string]any) rpcMessageKind {
	_, hasID := raw["id"]
	_, hasMethod := raw["method"]
	_, hasResult := raw["result"]
	_, hasError := raw["error"]

	switch {
	case hasID && hasMethod:
		return rpcMessageRequest
	case hasID && (hasResult || hasError):
		return rpcMessageResponse
	case hasMethod && !hasID:
		return rpcMessageNotification
	default:
		return rpcMessageUnknown
	}
}

// rpcIDToInt converts a JSON-RPC id (which may be a float64 from JSON decoding) to int.
func rpcIDToInt(v any) int {
	switch id := v.(type) {
	case float64:
		return int(id)
	case int:
		return id
	case json.Number:
		n, err := id.Int64()
		if err != nil {
			return 0
		}
		return int(n)
	default:
		return 0
	}
}

// marshalRPCRequest serializes a JSON-RPC request to a newline-delimited JSON line.
func marshalRPCRequest(id int, method string, params map[string]any) ([]byte, error) {
	msg := map[string]any{
		"id":     id,
		"method": method,
	}
	if params != nil {
		msg["params"] = params
	}
	b, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

// marshalRPCResponse serializes a JSON-RPC response to a newline-delimited JSON line.
func marshalRPCResponse(id any, result any) ([]byte, error) {
	msg := map[string]any{
		"id":     id,
		"result": result,
	}
	b, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

// marshalRPCError serializes a JSON-RPC error response to a newline-delimited JSON line.
func marshalRPCError(id any, code int, message string) ([]byte, error) {
	msg := map[string]any{
		"id": id,
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	}
	b, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}
