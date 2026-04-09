package llm

import "go-ops-agent/internal/config"

type Client struct {
	Config config.ProviderConfig
}

func NewClient(cfg config.ProviderConfig) *Client {
	return &Client{Config: cfg}
}
