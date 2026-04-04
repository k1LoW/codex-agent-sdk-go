package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	codex "github.com/k1LoW/codex-agent-sdk-go"
)

func main() {
	ctx := context.Background()

	client := codex.NewClient(
		codex.WithOnCommandApproval(func(_ context.Context, req codex.CommandApprovalRequest) (codex.ApprovalDecision, error) {
			fmt.Fprintf(os.Stderr, "[approval] command: %s\n", req.Command)
			return codex.DecisionAccept, nil
		}),
		codex.WithOnFileChangeApproval(func(_ context.Context, _ codex.FileChangeApprovalRequest) (codex.ApprovalDecision, error) {
			return codex.DecisionAccept, nil
		}),
	)

	if err := client.Connect(ctx); err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	thread, err := client.StartThread(ctx, codex.WithModel("o3"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprintf(os.Stderr, "[thread] %s started\n", thread.ID)

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		for evt, err := range client.StartTurn(ctx, thread.ID,
			[]codex.UserInput{codex.TextInput(input)}) {
			if err != nil {
				log.Fatal(err)
			}
			switch e := evt.(type) {
			case *codex.AgentMessageDeltaEvent:
				fmt.Print(e.Delta)
			case *codex.ItemCompletedEvent:
				if msg, ok := e.Item.(*codex.AgentMessageItem); ok {
					fmt.Println(msg.Text)
				}
				if cmd, ok := e.Item.(*codex.CommandExecutionItem); ok {
					fmt.Fprintf(os.Stderr, "[cmd] %s (exit=%v)\n", cmd.Command, cmd.ExitCode)
				}
			case *codex.TurnCompletedEvent:
				fmt.Println()
			}
		}
	}
}
