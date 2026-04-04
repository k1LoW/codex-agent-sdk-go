# codex-agent-sdk-go

An unofficial Go SDK for [OpenAI Codex](https://github.com/openai/codex).

It communicates with the Codex app-server via a subprocess (`codex app-server --listen stdio://`), supporting both one-shot queries and interactive bidirectional sessions with tool approval callbacks.

## Requirements

- Go 1.25+
- [Codex CLI](https://github.com/openai/codex) installed (`npm install -g @openai/codex`)

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"log"

	codex "github.com/k1LoW/codex-agent-sdk-go"
)

func main() {
	ctx := context.Background()
	for evt, err := range codex.Query(ctx, "What is 2 + 2?") {
		if err != nil {
			log.Fatal(err)
		}
		if e, ok := evt.(*codex.ItemCompletedEvent); ok {
			if msg, ok := e.Item.(*codex.AgentMessageItem); ok {
				fmt.Println(msg.Text)
			}
		}
	}
}
```

## Interactive Client

```go
client := codex.NewClient(
	codex.WithOnCommandApproval(func(_ context.Context, req codex.CommandApprovalRequest) (codex.ApprovalDecision, error) {
		fmt.Printf("Approve command: %s? ", req.Command)
		return codex.DecisionAccept, nil
	}),
)

if err := client.Connect(ctx); err != nil {
	log.Fatal(err)
}
defer client.Close()

thread, _ := client.StartThread(ctx, codex.WithModel("o3"))

for evt, err := range client.StartTurn(ctx, thread.ID,
	[]codex.UserInput{codex.TextInput("Hello")}) {
	if err != nil {
		log.Fatal(err)
	}
	switch e := evt.(type) {
	case *codex.AgentMessageDeltaEvent:
		fmt.Print(e.Delta)
	case *codex.TurnCompletedEvent:
		fmt.Println()
	}
}
```

## References

- [OpenAI Codex](https://github.com/openai/codex) - The original Codex CLI and SDK
- [claude-agent-sdk-go](https://github.com/k1LoW/claude-agent-sdk-go) - Claude Agent SDK for Go (architectural reference)
