package main

import (
	"context"
	"fmt"
	"log"
	"os"

	codex "github.com/k1LoW/codex-agent-sdk-go"
)

func main() {
	ctx := context.Background()
	prompt := "What is 2 + 2?"
	if len(os.Args) > 1 {
		prompt = os.Args[1]
	}

	for evt, err := range codex.Query(ctx, prompt,
		codex.WithOptions(
			codex.WithOnCommandApproval(func(_ context.Context, req codex.CommandApprovalRequest) (codex.ApprovalDecision, error) {
				fmt.Fprintf(os.Stderr, "[approval] command: %s\n", req.Command)
				return codex.DecisionAccept, nil
			}),
		),
	) {
		if err != nil {
			log.Fatal(err)
		}
		switch e := evt.(type) {
		case *codex.ItemCompletedEvent:
			if msg, ok := e.Item.(*codex.AgentMessageItem); ok {
				fmt.Println(msg.Text)
			}
		case *codex.AgentMessageDeltaEvent:
			fmt.Print(e.Delta)
		case *codex.TurnCompletedEvent:
			fmt.Fprintf(os.Stderr, "\n[done] turn %s completed\n", e.Turn.ID)
		}
	}
}
