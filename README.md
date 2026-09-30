# codex-agent-sdk-go

[![Go Reference](https://pkg.go.dev/badge/github.com/k1LoW/codex-agent-sdk-go.svg)](https://pkg.go.dev/github.com/k1LoW/codex-agent-sdk-go) ![Coverage](https://raw.githubusercontent.com/k1LoW/octocovs/main/badges/k1LoW/codex-agent-sdk-go/coverage.svg) ![Code to Test Ratio](https://raw.githubusercontent.com/k1LoW/octocovs/main/badges/k1LoW/codex-agent-sdk-go/ratio.svg) ![Test Execution Time](https://raw.githubusercontent.com/k1LoW/octocovs/main/badges/k1LoW/codex-agent-sdk-go/time.svg)

`codex-agent-sdk-go` is an **unofficial** Go SDK for [OpenAI Codex](https://github.com/openai/codex).

It communicates with the Codex app-server via a subprocess, supporting both one-shot queries and interactive bidirectional sessions.

## Usage

### Simple query

``` go
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

### With options

``` go
for evt, err := range codex.Query(ctx, "Hello",
	codex.WithOptions(
		codex.WithConfig(map[string]string{
			"model":           "gpt-5.4",
			"approval_policy": "on-request",
		}),
	),
	codex.WithThreadOptions(
		codex.WithModel("o3"),
		codex.WithApprovalPolicy("full-auto"),
	),
	codex.WithTurnOptions(
		codex.WithEffort("high"),
	),
) {
	// ...
}
```

### Interactive client

``` go
client := codex.NewClient()
if err := client.Connect(ctx); err != nil {
	log.Fatal(err)
}
defer client.Close()

// First turn
thread, _ := client.StartThread(ctx, codex.WithModel("o3"))

for evt, err := range client.StartTurn(ctx, thread.ID,
	[]codex.UserInput{codex.TextInput("What is the capital of France?")}) {
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

// Follow-up turn
for evt, err := range client.StartTurn(ctx, thread.ID,
	[]codex.UserInput{codex.TextInput("And Germany?")}) {
	// ...
}
```

### Command approval

``` go
client := codex.NewClient(
	codex.WithOnCommandApproval(func(_ context.Context, req codex.CommandApprovalRequest) (codex.ApprovalDecision, error) {
		fmt.Printf("Approve command: %s? ", req.Command)
		return codex.DecisionAccept, nil
	}),
	codex.WithOnFileChangeApproval(func(_ context.Context, req codex.FileChangeApprovalRequest) (codex.ApprovalDecision, error) {
		return codex.DecisionAccept, nil
	}),
)
```

### User input

``` go
client := codex.NewClient(
	codex.WithOnUserInput(func(_ context.Context, req codex.UserInputRequest) (map[string]string, error) {
		answers := make(map[string]string)
		for _, q := range req.Questions {
			id, _ := q["id"].(string)
			text, _ := q["text"].(string)
			fmt.Printf("%s: ", text)
			var answer string
			fmt.Scanln(&answer)
			answers[id] = answer
		}
		return answers, nil
	}),
)
```

### MCP elicitation

``` go
client := codex.NewClient(
	codex.WithOnElicitation(func(_ context.Context, req codex.ElicitationRequest) (codex.ElicitationResponse, error) {
		fmt.Printf("MCP server %s is requesting input\n", req.ServerName)
		return codex.ElicitationResponse{
			Action:   codex.ElicitationAccept,
			Response: map[string]any{"answer": "yes"},
		}, nil
	}),
)
```

## Prerequisites

- Go 1.25+
- [Codex CLI](https://github.com/openai/codex) installed and available in PATH

## References

- [OpenAI Codex](https://github.com/openai/codex) - The original Codex CLI and SDK
- [claude-agent-sdk-go](https://github.com/k1LoW/claude-agent-sdk-go) - Claude Agent SDK for Go (architectural reference)
