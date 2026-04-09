package llm

import (
	"context"
	"errors"
	"strings"

	openai "github.com/sashabaranov/go-openai"

	"go-ops-agent/internal/config"
	"go-ops-agent/internal/prompt"
)

type Client struct {
	Config config.ProviderConfig
	SDK    *openai.Client
}

func NewClient(cfg config.ProviderConfig) (*Client, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errors.New("missing API key, set OPS_AGENT_API_KEY or configure provider.api_key")
	}

	clientConfig := openai.DefaultConfig(cfg.APIKey)
	if strings.TrimSpace(cfg.BaseURL) != "" {
		clientConfig.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	}

	return &Client{
		Config: cfg,
		SDK:    openai.NewClientWithConfig(clientConfig),
	}, nil
}

func (c *Client) Chat(ctx context.Context, userPrompt string) (string, error) {
	resp, err := c.SDK.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: c.Config.Model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: prompt.SystemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userPrompt},
		},
		Temperature: 0.2,
	})
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", errors.New("empty response from LLM provider")
	}

	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}
