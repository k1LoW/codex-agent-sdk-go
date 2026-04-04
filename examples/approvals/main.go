package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	codex "github.com/k1LoW/codex-agent-sdk-go"
)

func main() {
	ctx := context.Background()

	client := codex.NewClient(
		codex.WithOnCommandApproval(func(_ context.Context, req codex.CommandApprovalRequest) (codex.ApprovalDecision, error) {
			// Deny dangerous commands.
			if strings.Contains(req.Command, "rm -rf") {
				fmt.Printf("[DENIED] dangerous command: %s\n", req.Command)
				return codex.DecisionDecline, nil
			}
			fmt.Printf("[APPROVED] command: %s\n", req.Command)
			return codex.DecisionAccept, nil
		}),
		codex.WithOnFileChangeApproval(func(_ context.Context, req codex.FileChangeApprovalRequest) (codex.ApprovalDecision, error) {
			fmt.Printf("[APPROVED] file change (item=%s)\n", req.ItemID)
			return codex.DecisionAccept, nil
		}),
		codex.WithOnToolCall(func(_ context.Context, req codex.ToolCallRequest) (codex.ToolCallResponse, error) {
			fmt.Printf("[TOOL CALL] %s(%v)\n", req.Tool, req.Arguments)
			return codex.ToolCallResponse{
				ContentItems: []map[string]any{{"type": "text", "text": "tool result"}},
				Success:      true,
			}, nil
		}),
	)

	if err := client.Connect(ctx); err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	thread, err := client.StartThread(ctx)
	if err != nil {
		log.Fatal(err)
	}

	for evt, err := range client.StartTurn(ctx, thread.ID,
		[]codex.UserInput{codex.TextInput("List files in the current directory")}) {
		if err != nil {
			log.Fatal(err)
		}
		switch e := evt.(type) {
		case *codex.ItemCompletedEvent:
			if msg, ok := e.Item.(*codex.AgentMessageItem); ok {
				fmt.Println(msg.Text)
			}
		case *codex.TurnCompletedEvent:
			fmt.Println("[turn completed]")
		}
	}
}
