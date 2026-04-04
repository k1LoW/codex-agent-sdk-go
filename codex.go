// Package codex provides a Go SDK for OpenAI Codex.
//
// It communicates with the Codex app-server via a subprocess, supporting
// both one-shot queries and interactive bidirectional sessions.
//
// # Quick Start
//
//	for evt, err := range codex.Query(ctx, "What is 2 + 2?") {
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//	    if e, ok := evt.(*codex.ItemCompletedEvent); ok {
//	        if msg, ok := e.Item.(*codex.AgentMessageItem); ok {
//	            fmt.Print(msg.Text)
//	        }
//	    }
//	}
//
// # Interactive Client
//
//	client := codex.NewClient()
//	if err := client.Connect(ctx); err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
//	thread, _ := client.StartThread(ctx, codex.WithModel("o3"))
//	for evt, err := range client.StartTurn(ctx, thread.ID,
//	    []codex.UserInput{codex.TextInput("Hello")}) {
//	    // ...
//	}
package codex

import (
	"context"
	"fmt"
	"iter"

	"github.com/k1LoW/codex-agent-sdk-go/version"
)

// Query performs a one-shot query to the Codex agent and returns an iterator
// over the response events. The iterator handles all setup and cleanup
// automatically.
func Query(ctx context.Context, prompt string, opts ...QueryOption) iter.Seq2[Event, error] {
	return func(yield func(Event, error) bool) {
		qopts := applyQueryOptions(opts)
		options := applyOptions(qopts.codex)

		transport := newSubprocessTransport(options)
		if err := transport.Connect(ctx); err != nil {
			yield(nil, err)
			return
		}

		sess := newSession(ctx, transport, options)
		sess.start()
		defer sess.close()

		// 1. Initialize.
		_, err := sess.sendRequest(ctx, "initialize", map[string]any{
			"clientInfo": map[string]any{
				"name":    version.Name,
				"version": version.Version,
			},
			"capabilities": map[string]any{
				"experimentalApi": true,
			},
		})
		if err != nil {
			yield(nil, fmt.Errorf("initialize failed: %w", err))
			return
		}

		// 2. thread/start.
		threadOpts := applyThreadOptions(qopts.thread)
		threadResult, err := sess.sendRequest(ctx, "thread/start", threadOpts.toParams())
		if err != nil {
			yield(nil, fmt.Errorf("thread/start failed: %w", err))
			return
		}

		threadMap, ok := threadResult.(map[string]any)
		if !ok {
			yield(nil, fmt.Errorf("unexpected thread/start response"))
			return
		}
		threadRaw, ok := threadMap["thread"].(map[string]any)
		if !ok {
			yield(nil, fmt.Errorf("missing thread in response"))
			return
		}
		threadID := strVal(threadRaw, "id")

		// 3. turn/start.
		turnOpts := applyTurnOptions(qopts.turn)
		params := buildTurnStartParams(threadID, []UserInput{TextInput(prompt)}, turnOpts)
		_, err = sess.sendRequest(ctx, "turn/start", params)
		if err != nil {
			yield(nil, fmt.Errorf("turn/start failed: %w", err))
			return
		}

		// 4. Yield events until turn/completed.
		for {
			select {
			case evt, ok := <-sess.eventCh:
				if !ok {
					if readErr := sess.readError(); readErr != nil {
						yield(nil, readErr)
					}
					return
				}
				if !yield(evt, nil) {
					return
				}
				if _, isTurnCompleted := evt.(*TurnCompletedEvent); isTurnCompleted {
					return
				}
			case <-ctx.Done():
				yield(nil, ctx.Err())
				return
			}
		}
	}
}
