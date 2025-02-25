package main

import (
	"context"
	"fmt"
	"os"

	"example.com/web-app-creator/agents"
	"example.com/web-app-creator/llm"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	ctx := context.TODO()
	openAIAPIKey := os.Getenv("OPENAI_API_KEY")
	provider := llm.NewOpenAIProvider(openAIAPIKey, "")

	agent := agents.Agent{Name: "agent", SystemPrompt: "You always answer as angry old man.", Model: "gpt-4-turbo-2024-04-09", Llm: provider}
	ans, _ := agent.Chat(ctx, "pożycz na piwo")

	fmt.Println(ans)
}
