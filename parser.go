package codex

import "fmt"

// parseEvent converts a server notification into a typed Event.
func parseEvent(method string, params map[string]any) (Event, error) {
	switch method {
	case "thread/started":
		return parseThreadStartedEvent(params)
	case "turn/started":
		return parseTurnStartedEvent(params)
	case "turn/completed":
		return parseTurnCompletedEvent(params)
	case "item/started":
		return parseItemEvent(params, func(threadID, turnID string, item ThreadItem) Event {
			return &ItemStartedEvent{ThreadID: threadID, TurnID: turnID, Item: item}
		})
	case "item/completed":
		return parseItemEvent(params, func(threadID, turnID string, item ThreadItem) Event {
			return &ItemCompletedEvent{ThreadID: threadID, TurnID: turnID, Item: item}
		})
	case "item/agentMessage/delta":
		return &AgentMessageDeltaEvent{
			ThreadID: strVal(params, "threadId"),
			TurnID:   strVal(params, "turnId"),
			ItemID:   strVal(params, "itemId"),
			Delta:    strVal(params, "delta"),
		}, nil
	case "item/commandExecution/outputDelta":
		return &CommandOutputDeltaEvent{
			ThreadID: strVal(params, "threadId"),
			TurnID:   strVal(params, "turnId"),
			ItemID:   strVal(params, "itemId"),
			Delta:    strVal(params, "delta"),
		}, nil
	case "item/reasoning/textDelta":
		return &ReasoningDeltaEvent{
			ThreadID:     strVal(params, "threadId"),
			TurnID:       strVal(params, "turnId"),
			ItemID:       strVal(params, "itemId"),
			Delta:        strVal(params, "delta"),
			ContentIndex: intVal(params, "contentIndex"),
		}, nil
	case "error":
		return &ErrorEvent{
			ThreadID: strVal(params, "threadId"),
			TurnID:   strVal(params, "turnId"),
			Code:     intVal(params, "code"),
			Message:  strVal(params, "message"),
			Details:  strVal(params, "details"),
		}, nil
	default:
		// Unknown notification types are silently ignored.
		return nil, nil
	}
}

func parseThreadStartedEvent(params map[string]any) (*ThreadStartedEvent, error) {
	evt := &ThreadStartedEvent{
		ThreadID: strVal(params, "threadId"),
	}
	if threadRaw, ok := params["thread"].(map[string]any); ok {
		evt.Thread = parseThread(threadRaw)
	}
	return evt, nil
}

func parseTurnStartedEvent(params map[string]any) (*TurnStartedEvent, error) {
	evt := &TurnStartedEvent{
		ThreadID: strVal(params, "threadId"),
	}
	if turnRaw, ok := params["turn"].(map[string]any); ok {
		evt.Turn = parseTurn(turnRaw)
	}
	return evt, nil
}

func parseTurnCompletedEvent(params map[string]any) (*TurnCompletedEvent, error) {
	evt := &TurnCompletedEvent{
		ThreadID: strVal(params, "threadId"),
	}
	if turnRaw, ok := params["turn"].(map[string]any); ok {
		evt.Turn = parseTurn(turnRaw)
	}
	return evt, nil
}

func parseItemEvent(params map[string]any, mk func(string, string, ThreadItem) Event) (Event, error) {
	threadID := strVal(params, "threadId")
	turnID := strVal(params, "turnId")
	itemRaw, ok := params["item"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing item in event")
	}
	item, err := parseThreadItem(itemRaw)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, nil
	}
	return mk(threadID, turnID, item), nil
}

// parseThread converts a raw JSON map to a Thread.
func parseThread(raw map[string]any) *Thread {
	t := &Thread{
		ID:            strVal(raw, "id"),
		Status:        strVal(raw, "status"),
		CWD:           strVal(raw, "cwd"),
		Model:         strVal(raw, "model"),
		ModelProvider: strVal(raw, "modelProvider"),
		CreatedAt:     floatVal(raw, "createdAt"),
		UpdatedAt:     floatVal(raw, "updatedAt"),
		Ephemeral:     boolVal(raw, "ephemeral"),
	}
	if turnsRaw, ok := raw["turns"].([]any); ok {
		for _, tr := range turnsRaw {
			if turnMap, ok := tr.(map[string]any); ok {
				t.Turns = append(t.Turns, *parseTurn(turnMap))
			}
		}
	}
	return t
}

// parseTurn converts a raw JSON map to a Turn.
func parseTurn(raw map[string]any) *Turn {
	t := &Turn{
		ID:     strVal(raw, "id"),
		Status: TurnStatus(strVal(raw, "status")),
	}
	if errRaw, ok := raw["error"].(map[string]any); ok {
		t.Error = &TurnErrorInfo{
			Message: strVal(errRaw, "message"),
			Details: strVal(errRaw, "additionalDetails"),
		}
	}
	if itemsRaw, ok := raw["items"].([]any); ok {
		for _, ir := range itemsRaw {
			if itemMap, ok := ir.(map[string]any); ok {
				item, err := parseThreadItem(itemMap)
				if err == nil && item != nil {
					t.Items = append(t.Items, item)
				}
			}
		}
	}
	return t
}

// parseThreadItem converts a raw JSON map to a typed ThreadItem.
func parseThreadItem(raw map[string]any) (ThreadItem, error) {
	typ := strVal(raw, "type")
	id := strVal(raw, "id")

	switch typ {
	case "agentMessage":
		return &AgentMessageItem{
			ID:   id,
			Text: strVal(raw, "text"),
		}, nil
	case "plan":
		return &PlanItem{
			ID:   id,
			Text: strVal(raw, "text"),
		}, nil
	case "reasoning":
		return &ReasoningItem{
			ID:      id,
			Summary: strSlice(raw, "summary"),
			Content: strSlice(raw, "content"),
		}, nil
	case "commandExecution":
		item := &CommandExecutionItem{
			ID:               id,
			Command:          strVal(raw, "command"),
			CWD:              strVal(raw, "cwd"),
			Status:           CommandExecutionStatus(strVal(raw, "status")),
			AggregatedOutput: strVal(raw, "aggregatedOutput"),
		}
		if ec, ok := intPtr(raw, "exitCode"); ok {
			item.ExitCode = ec
		}
		if d, ok := intPtr(raw, "durationMs"); ok {
			item.DurationMs = d
		}
		return item, nil
	case "fileChange":
		item := &FileChangeItem{
			ID:     id,
			Status: PatchApplyStatus(strVal(raw, "status")),
		}
		if changesRaw, ok := raw["changes"].([]any); ok {
			for _, cr := range changesRaw {
				if cm, ok := cr.(map[string]any); ok {
					item.Changes = append(item.Changes, FileUpdateChange{
						Path: strVal(cm, "path"),
						Kind: strVal(cm, "kind"),
					})
				}
			}
		}
		return item, nil
	case "mcpToolCall":
		item := &McpToolCallItem{
			ID:        id,
			Server:    strVal(raw, "server"),
			Tool:      strVal(raw, "tool"),
			Arguments: raw["arguments"],
			Result:    raw["result"],
			Status:    McpToolCallStatus(strVal(raw, "status")),
		}
		if errRaw, ok := raw["error"].(map[string]any); ok {
			item.Error = &McpToolCallError{Message: strVal(errRaw, "message")}
		}
		if d, ok := intPtr(raw, "durationMs"); ok {
			item.DurationMs = d
		}
		return item, nil
	case "dynamicToolCall":
		item := &DynamicToolCallItem{
			ID:        id,
			Tool:      strVal(raw, "tool"),
			Arguments: raw["arguments"],
			Status:    DynamicToolCallStatus(strVal(raw, "status")),
		}
		if ci, ok := raw["contentItems"].([]any); ok {
			for _, c := range ci {
				if cm, ok := c.(map[string]any); ok {
					item.ContentItems = append(item.ContentItems, cm)
				}
			}
		}
		if s, ok := raw["success"].(bool); ok {
			item.Success = &s
		}
		if d, ok := intPtr(raw, "durationMs"); ok {
			item.DurationMs = d
		}
		return item, nil
	case "webSearch":
		return &WebSearchItem{
			ID:    id,
			Query: strVal(raw, "query"),
		}, nil
	case "userMessage":
		item := &UserMessageItem{ID: id}
		if contentRaw, ok := raw["content"].([]any); ok {
			for _, cr := range contentRaw {
				if cm, ok := cr.(map[string]any); ok {
					item.Content = append(item.Content, UserInput{
						Type: strVal(cm, "type"),
						Text: strVal(cm, "text"),
						Path: strVal(cm, "path"),
					})
				}
			}
		}
		return item, nil
	case "error":
		return &ErrorItem{
			ID:      id,
			Message: strVal(raw, "message"),
		}, nil
	default:
		// Unknown item types are silently skipped.
		return nil, nil
	}
}

// Helper functions for extracting typed values from map[string]any.

func strVal(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func intVal(m map[string]any, key string) int {
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	default:
		return 0
	}
}

func floatVal(m map[string]any, key string) float64 {
	if v, ok := m[key].(float64); ok {
		return v
	}
	return 0
}

func boolVal(m map[string]any, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

func intPtr(m map[string]any, key string) (*int, bool) {
	v, ok := m[key]
	if !ok || v == nil {
		return nil, false
	}
	switch n := v.(type) {
	case float64:
		i := int(n)
		return &i, true
	case int:
		return &n, true
	default:
		return nil, false
	}
}

func strSlice(m map[string]any, key string) []string {
	raw, ok := m[key].([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result
}
