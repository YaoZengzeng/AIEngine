package proxy

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
)

type Proxy interface {
	Proxy(host string, provider string, model string, message string) (string, error)
}

type proxyImpl struct {
}

func NewProxy() (Proxy, error) {
	return &proxyImpl{}, nil
}

func (p *proxyImpl) Proxy(host string, provider string, model string, message string) (string, error) {
	// Only support ollama models now.
	switch provider {
	case "ollama":
		return forwardToOllama(host, model, message)
	default:
		return "", fmt.Errorf("provider %s is not supported yet", provider)
	}
	return "", nil
}

func forwardToOllama(host string, model string, message string) (string, error) {
	llm, err := ollama.New(ollama.WithModel(model), ollama.WithServerURL(host))
	if err != nil {
		return "", err
	}
	ctx := context.Background()

	content := []llms.MessageContent{
		// llms.TextParts(llms.ChatMessageTypeSystem, "You are a company branding design wizard."),
		llms.TextParts(llms.ChatMessageTypeHuman, message),
	}

	var resp string
	completion, err := llm.GenerateContent(ctx, content, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
		resp += string(chunk)
		return nil
	}))
	if err != nil {
		return "", err
	}
	_ = completion

	return resp, nil
}
