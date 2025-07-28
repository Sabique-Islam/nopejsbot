package ai

import (
	"context"
	"fmt"
	"os"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

func ExplainDiff(ctx context.Context, userPrompt string, diff string) (string, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GEMINI_API_KEY not set")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return "", fmt.Errorf("gemini client init error: %w", err)
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-2.5-flash")

	prompt := fmt.Sprintf(
		`You are an AI assistant helping review GitHub pull requests.
		 A user has commented: "%s"

		 Please analyze and respond based on the following Git diff:
		 %s`, userPrompt, diff)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("gemini generation error: %w", err)
	}

	var result string
	for _, candidate := range resp.Candidates {
		for _, part := range candidate.Content.Parts {
			if text, ok := part.(genai.Text); ok {
				result += string(text) + "\n"
			}
		}
	}
	if result == "" {
		return "", fmt.Errorf("no content returned by gemini")
	}
	return result, nil
}
