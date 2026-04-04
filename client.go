package codex

import (
	"context"
	"fmt"
	"iter"

	"github.com/k1LoW/codex-agent-sdk-go/version"
)

// Client provides an interactive multi-turn API for communicating with the Codex agent.
type Client struct {
	options   *Options
	transport Transport
	sess      *session
}

// NewClient creates a new Client instance.
func NewClient(opts ...Option) *Client {
	return &Client{
		options: applyOptions(opts),
	}
}

// Connect launches the codex app-server subprocess and sends the initialize request.
func (c *Client) Connect(ctx context.Context) error {
	t := newSubprocessTransport(c.options)
	if err := t.Connect(ctx); err != nil {
		return err
	}
	c.transport = t

	c.sess = newSession(ctx, c.transport, c.options)
	c.sess.start()

	// Send initialize request.
	_, err := c.sess.sendRequest(ctx, "initialize", map[string]any{
		"clientInfo": map[string]any{
			"name":    version.Name,
			"version": version.Version,
		},
		"capabilities": map[string]any{
			"experimentalApi": true,
		},
	})
	if err != nil {
		c.sess.close()
		return fmt.Errorf("initialize failed: %w", err)
	}

	return nil
}

// StartThread sends thread/start and returns the created Thread.
func (c *Client) StartThread(ctx context.Context, opts ...ThreadOption) (*Thread, error) {
	if c.sess == nil {
		return nil, fmt.Errorf("client not connected")
	}
	threadOpts := applyThreadOptions(opts)
	params := threadOpts.toParams()

	result, err := c.sess.sendRequest(ctx, "thread/start", params)
	if err != nil {
		return nil, fmt.Errorf("thread/start failed: %w", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected thread/start response type: %T", result)
	}

	threadRaw, ok := resultMap["thread"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing thread in thread/start response")
	}

	thread := parseThread(threadRaw)

	// Apply top-level response fields.
	if model, ok := resultMap["model"].(string); ok && thread.Model == "" {
		thread.Model = model
	}
	if mp, ok := resultMap["modelProvider"].(string); ok && thread.ModelProvider == "" {
		thread.ModelProvider = mp
	}

	return thread, nil
}

// ResumeThread sends thread/resume and returns the resumed Thread.
func (c *Client) ResumeThread(ctx context.Context, threadID string, opts ...ThreadOption) (*Thread, error) {
	if c.sess == nil {
		return nil, fmt.Errorf("client not connected")
	}
	threadOpts := applyThreadOptions(opts)
	params := threadOpts.toParams()
	params["threadId"] = threadID

	result, err := c.sess.sendRequest(ctx, "thread/resume", params)
	if err != nil {
		return nil, fmt.Errorf("thread/resume failed: %w", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected thread/resume response type: %T", result)
	}

	threadRaw, ok := resultMap["thread"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing thread in thread/resume response")
	}

	return parseThread(threadRaw), nil
}

// StartTurn sends turn/start and returns an iterator of Events for the turn.
// The iterator completes when a TurnCompletedEvent is received or the session ends.
func (c *Client) StartTurn(ctx context.Context, threadID string, input []UserInput, opts ...TurnOption) iter.Seq2[Event, error] {
	return func(yield func(Event, error) bool) {
		if c.sess == nil {
			yield(nil, fmt.Errorf("client not connected"))
			return
		}
		turnOpts := applyTurnOptions(opts)
		params := buildTurnStartParams(threadID, input, turnOpts)

		_, err := c.sess.sendRequest(ctx, "turn/start", params)
		if err != nil {
			yield(nil, fmt.Errorf("turn/start failed: %w", err))
			return
		}

		for {
			select {
			case evt, ok := <-c.sess.eventCh:
				if !ok {
					if readErr := c.sess.readError(); readErr != nil {
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

// Interrupt sends turn/interrupt to stop an active turn.
func (c *Client) Interrupt(ctx context.Context, threadID string, turnID string) error {
	if c.sess == nil {
		return fmt.Errorf("client not connected")
	}
	_, err := c.sess.sendRequest(ctx, "turn/interrupt", map[string]any{
		"threadId": threadID,
		"turnId":   turnID,
	})
	return err
}

// Close terminates the session and releases resources.
func (c *Client) Close() error {
	if c.sess != nil {
		c.sess.close()
	}
	return nil
}
