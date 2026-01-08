package internal

import (
	"context"
	"fmt"

	openai "github.com/sashabaranov/go-openai"

	"github.com/ringecosystem/degov-square/internal/config"
)

// OpenRouterClient handles AI API calls using OpenAI-compatible SDK
type OpenRouterClient struct {
	client *openai.Client
	model  string
}

// NewOpenRouterClient creates a new OpenRouter client
func NewOpenRouterClient() *OpenRouterClient {
	cfg := config.GetConfig()
	apiKey := cfg.GetString("OPENROUTER_API_KEY")
	model := cfg.GetStringWithDefault("OPENROUTER_MODEL", "google/gemini-2.0-flash-001")

	openaiConfig := openai.DefaultConfig(apiKey)
	openaiConfig.BaseURL = "https://openrouter.ai/api/v1"

	return &OpenRouterClient{
		client: openai.NewClientWithConfig(openaiConfig),
		model:  model,
	}
}

// ChatCompletionRequest represents a chat completion request
type ChatCompletionRequest struct {
	SystemPrompt string
	UserPrompt   string
	Temperature  float32
	MaxTokens    int
}

// ChatCompletion sends a chat completion request and returns the response content
func (c *OpenRouterClient) ChatCompletion(ctx context.Context, req ChatCompletionRequest) (string, error) {
	// Set defaults
	if req.Temperature == 0 {
		req.Temperature = 0.3
	}
	if req.MaxTokens == 0 {
		req.MaxTokens = 4096
	}

	resp, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: c.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: req.SystemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: req.UserPrompt},
		},
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create chat completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return resp.Choices[0].Message.Content, nil
}

// GetModel returns the configured model name
func (c *OpenRouterClient) GetModel() string {
	return c.model
}
