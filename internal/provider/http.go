package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ryewmn/agent-gateway-go/internal/model"
)

// HTTPProvider connects to an OpenAI-compatible upstream. API keys are read
// from configuration at startup and are never included in errors or logs.
type HTTPProvider struct {
	name, endpoint, apiKey string
	client *http.Client
}

func NewHTTP(name, endpoint, apiKey string, client *http.Client) *HTTPProvider {
	if client == nil { client = http.DefaultClient }
	return &HTTPProvider{name: name, endpoint: endpoint, apiKey: apiKey, client: client}
}

func (p *HTTPProvider) Name() string { return p.name }

func (p *HTTPProvider) Chat(ctx context.Context, input model.ChatRequest) (model.ChatResponse, error) {
	body, err := json.Marshal(input)
	if err != nil { return model.ChatResponse{}, fmt.Errorf("encode request: %w", err) }
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(body))
	if err != nil { return model.ChatResponse{}, fmt.Errorf("create request: %w", err) }
	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" { req.Header.Set("Authorization", "Bearer "+p.apiKey) }
	res, err := p.client.Do(req)
	if err != nil { return model.ChatResponse{}, fmt.Errorf("upstream request failed: %v: %w", err, ErrUnavailable) }
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 4<<20))
		if res.StatusCode == http.StatusTooManyRequests { return model.ChatResponse{}, ErrRateLimited }
		if res.StatusCode >= 500 { return model.ChatResponse{}, ErrUnavailable }
		return model.ChatResponse{}, fmt.Errorf("upstream returned status %d", res.StatusCode)
	}
	limited, err := io.ReadAll(io.LimitReader(res.Body, (4<<20)+1))
	if err != nil { return model.ChatResponse{}, fmt.Errorf("read response: %w", err) }
	if len(limited) > 4<<20 { return model.ChatResponse{}, fmt.Errorf("upstream response exceeds 4 MiB") }
	var out model.ChatResponse
	if err := json.Unmarshal(limited, &out); err != nil { return out, fmt.Errorf("decode response: %w", err) }
	return out, nil
}
