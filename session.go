package codex

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
)

type rpcResult struct {
	result any
	err    error
}

// session handles bidirectional JSON-RPC communication with codex app-server.
type session struct {
	ctx    context.Context //nostyle:contexts // readLoop goroutine needs the lifecycle context to detect cancellation
	cancel context.CancelFunc

	transport Transport
	options   *Options

	eventCh chan Event
	doneCh  chan struct{}

	mu                sync.Mutex
	pendingRequests   map[int]chan rpcResult
	bufferedResponses map[int]rpcResult // responses that arrived before sendRequest registered
	requestCounter    int

	readErrMu sync.Mutex
	readErr   error
	closed    atomic.Bool
}

func newSession(ctx context.Context, transport Transport, options *Options) *session {
	ctx, cancel := context.WithCancel(ctx)
	return &session{
		ctx:               ctx,
		cancel:            cancel,
		transport:         transport,
		options:           options,
		eventCh:           make(chan Event, 128),
		doneCh:            make(chan struct{}),
		pendingRequests:   make(map[int]chan rpcResult),
		bufferedResponses: make(map[int]rpcResult),
	}
}

func (s *session) start() {
	go s.readLoop()
}

func (s *session) readLoop() {
	defer close(s.doneCh)
	defer close(s.eventCh)

	for {
		raw, err := s.transport.ReadMessage()
		if err != nil {
			if !errors.Is(err, io.EOF) && !s.closed.Load() {
				s.setReadErr(err)
			}
			return
		}

		kind := classifyMessage(raw)
		switch kind {
		case rpcMessageResponse:
			s.handleResponse(raw)
		case rpcMessageRequest:
			// Server-initiated request (approval, tool call, etc.).
			go s.handleServerRequest(raw)
		case rpcMessageNotification:
			s.handleNotification(raw)
		default:
			// Unknown message shape; skip.
		}
	}
}

func (s *session) handleResponse(raw map[string]any) {
	id := rpcIDToInt(raw["id"])

	var result rpcResult
	if errObj, hasErr := raw["error"].(map[string]any); hasErr {
		result = rpcResult{err: &RPCError{
			SDKError: SDKError{Message: strVal(errObj, "message")},
			Code:     intVal(errObj, "code"),
			Data:     errObj["data"],
		}}
	} else {
		result = rpcResult{result: raw["result"]}
	}

	s.mu.Lock()
	ch, ok := s.pendingRequests[id]
	if ok {
		delete(s.pendingRequests, id)
		s.mu.Unlock()
		ch <- result
	} else {
		// Response arrived before sendRequest registered; buffer it.
		s.bufferedResponses[id] = result
		s.mu.Unlock()
	}
}

func (s *session) handleNotification(raw map[string]any) {
	method := strVal(raw, "method")
	params, _ := raw["params"].(map[string]any)
	if params == nil {
		params = map[string]any{}
	}

	evt, err := parseEvent(method, params)
	if err != nil || evt == nil {
		return
	}

	select {
	case s.eventCh <- evt:
	case <-s.ctx.Done():
	}
}

func (s *session) handleServerRequest(raw map[string]any) {
	id := raw["id"]
	method := strVal(raw, "method")
	params, _ := raw["params"].(map[string]any)
	if params == nil {
		params = map[string]any{}
	}

	var result any
	var err error

	switch method {
	case "item/commandExecution/requestApproval":
		result, err = s.handleCommandApproval(params)
	case "item/fileChange/requestApproval":
		result, err = s.handleFileChangeApproval(params)
	case "item/permissions/requestApproval":
		result, err = s.handlePermissionApproval(params)
	case "item/tool/call":
		result, err = s.handleToolCall(params)
	case "item/tool/requestUserInput":
		result, err = s.handleUserInput(params)
	case "mcpServer/elicitation/request":
		result, err = s.handleElicitation(params)
	default:
		// Unknown server request: reject.
		s.sendRPCError(id, -32601, fmt.Sprintf("unsupported server request: %s", method))
		return
	}

	if err != nil {
		s.sendRPCError(id, -32603, err.Error())
		return
	}
	s.sendRPCResponse(id, result)
}

func (s *session) handleCommandApproval(params map[string]any) (any, error) {
	if s.options.OnCommandApproval == nil {
		return map[string]any{"decision": string(DecisionDecline)}, nil
	}
	req := CommandApprovalRequest{
		ThreadID: strVal(params, "threadId"),
		TurnID:   strVal(params, "turnId"),
		ItemID:   strVal(params, "itemId"),
		Command:  strVal(params, "command"),
		CWD:      strVal(params, "cwd"),
		Reason:   strVal(params, "reason"),
		Raw:      params,
	}
	decision, err := s.options.OnCommandApproval(s.ctx, req)
	if err != nil {
		return nil, err
	}
	return map[string]any{"decision": string(decision)}, nil
}

func (s *session) handleFileChangeApproval(params map[string]any) (any, error) {
	if s.options.OnFileChangeApproval == nil {
		return map[string]any{"decision": string(DecisionDecline)}, nil
	}
	req := FileChangeApprovalRequest{
		ThreadID: strVal(params, "threadId"),
		TurnID:   strVal(params, "turnId"),
		ItemID:   strVal(params, "itemId"),
		Reason:   strVal(params, "reason"),
		Raw:      params,
	}
	decision, err := s.options.OnFileChangeApproval(s.ctx, req)
	if err != nil {
		return nil, err
	}
	return map[string]any{"decision": string(decision)}, nil
}

func (s *session) handlePermissionApproval(params map[string]any) (any, error) {
	if s.options.OnPermissionApproval == nil {
		return map[string]any{"decision": string(DecisionDecline)}, nil
	}
	req := PermissionApprovalRequest{
		ThreadID: strVal(params, "threadId"),
		TurnID:   strVal(params, "turnId"),
		Raw:      params,
	}
	decision, err := s.options.OnPermissionApproval(s.ctx, req)
	if err != nil {
		return nil, err
	}
	return map[string]any{"decision": string(decision)}, nil
}

func (s *session) handleToolCall(params map[string]any) (any, error) {
	if s.options.OnToolCall == nil {
		return map[string]any{"contentItems": []any{}, "success": false}, nil
	}
	args, _ := params["arguments"].(map[string]any)
	req := ToolCallRequest{
		ThreadID:  strVal(params, "threadId"),
		TurnID:    strVal(params, "turnId"),
		CallID:    strVal(params, "callId"),
		Tool:      strVal(params, "tool"),
		Arguments: args,
	}
	resp, err := s.options.OnToolCall(s.ctx, req)
	if err != nil {
		return nil, err
	}
	items := make([]any, len(resp.ContentItems))
	for i, ci := range resp.ContentItems {
		items[i] = ci
	}
	return map[string]any{"contentItems": items, "success": resp.Success}, nil
}

func (s *session) handleUserInput(params map[string]any) (any, error) {
	if s.options.OnUserInput == nil {
		return map[string]any{"answers": map[string]any{}}, nil
	}
	var questions []map[string]any
	if qs, ok := params["questions"].([]any); ok {
		for _, q := range qs {
			if qm, ok := q.(map[string]any); ok {
				questions = append(questions, qm)
			}
		}
	}
	req := UserInputRequest{
		ThreadID:  strVal(params, "threadId"),
		TurnID:    strVal(params, "turnId"),
		ItemID:    strVal(params, "itemId"),
		Questions: questions,
	}
	answers, err := s.options.OnUserInput(s.ctx, req)
	if err != nil {
		return nil, err
	}
	// Convert map[string]string to map[string]any for JSON.
	answerMap := make(map[string]any, len(answers))
	for k, v := range answers {
		answerMap[k] = v
	}
	return map[string]any{"answers": answerMap}, nil
}

func (s *session) handleElicitation(params map[string]any) (any, error) {
	if s.options.OnElicitation == nil {
		return map[string]any{"action": string(ElicitationCancel)}, nil
	}
	req := ElicitationRequest{
		ThreadID:   strVal(params, "threadId"),
		TurnID:     strVal(params, "turnId"),
		ServerName: strVal(params, "serverName"),
		Raw:        params,
	}
	resp, err := s.options.OnElicitation(s.ctx, req)
	if err != nil {
		return nil, err
	}
	result := map[string]any{"action": string(resp.Action)}
	if resp.Response != nil {
		result["response"] = resp.Response
	}
	return result, nil
}

// sendRequest sends a JSON-RPC request and waits for the response.
func (s *session) sendRequest(ctx context.Context, method string, params map[string]any) (any, error) {
	s.mu.Lock()
	s.requestCounter++
	id := s.requestCounter

	// Check if the response already arrived (buffered by readLoop).
	if buffered, ok := s.bufferedResponses[id]; ok {
		delete(s.bufferedResponses, id)
		s.mu.Unlock()
		// Still need to send the request so the server sees it,
		// but the response is already available. However, this case means
		// the response arrived before the request was even sent, which
		// should not happen in normal operation. Just return the buffered result.
		return buffered.result, buffered.err
	}

	ch := make(chan rpcResult, 1)
	s.pendingRequests[id] = ch
	s.mu.Unlock()

	data, err := marshalRPCRequest(id, method, params)
	if err != nil {
		s.mu.Lock()
		delete(s.pendingRequests, id)
		s.mu.Unlock()
		return nil, err
	}

	if err := s.transport.Write(data); err != nil {
		s.mu.Lock()
		delete(s.pendingRequests, id)
		s.mu.Unlock()
		return nil, err
	}

	// Check again if response arrived between registering and writing.
	s.mu.Lock()
	if buffered, ok := s.bufferedResponses[id]; ok {
		delete(s.bufferedResponses, id)
		delete(s.pendingRequests, id)
		s.mu.Unlock()
		return buffered.result, buffered.err
	}
	s.mu.Unlock()

	select {
	case result := <-ch:
		return result.result, result.err
	case <-ctx.Done():
		s.mu.Lock()
		delete(s.pendingRequests, id)
		s.mu.Unlock()
		return nil, ctx.Err()
	case <-s.doneCh:
		// Final check for buffered response before giving up.
		s.mu.Lock()
		if buffered, ok := s.bufferedResponses[id]; ok {
			delete(s.bufferedResponses, id)
			s.mu.Unlock()
			return buffered.result, buffered.err
		}
		s.mu.Unlock()
		return nil, fmt.Errorf("session closed while waiting for response to %s", method)
	}
}

func (s *session) sendRPCResponse(id any, result any) {
	data, err := marshalRPCResponse(id, result)
	if err != nil {
		return
	}
	s.transport.Write(data) //nolint:errcheck // best-effort response to server; transport may already be closing
}

func (s *session) sendRPCError(id any, code int, message string) {
	data, err := marshalRPCError(id, code, message)
	if err != nil {
		return
	}
	s.transport.Write(data) //nolint:errcheck // best-effort response to server; transport may already be closing
}

func (s *session) setReadErr(err error) {
	s.readErrMu.Lock()
	defer s.readErrMu.Unlock()
	if s.readErr == nil {
		s.readErr = err
	}
}

func (s *session) readError() error {
	s.readErrMu.Lock()
	defer s.readErrMu.Unlock()
	return s.readErr
}

func (s *session) close() {
	s.closed.Store(true)
	s.cancel()
	s.transport.Close()
	// Wait for readLoop to finish.
	<-s.doneCh
}
