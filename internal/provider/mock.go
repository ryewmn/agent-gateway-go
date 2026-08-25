package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ryewmn/agent-gateway-go/internal/model"
)

// MockConfig makes failures and latency deterministic, which keeps CI repeatable.
type MockConfig struct {
	Name        string
	Latency     time.Duration
	FailEvery   uint64
	FailFirst   uint64
	ToolQuality int // 0-100; a stable request hash decides whether a call is correct.
}

type Mock struct {
	cfg   MockConfig
	calls atomic.Uint64
}

func NewMock(cfg MockConfig) *Mock {
	if cfg.Name == "" { cfg.Name = "mock" }
	if cfg.ToolQuality == 0 { cfg.ToolQuality = 100 }
	return &Mock{cfg: cfg}
}

func (m *Mock) Name() string { return m.cfg.Name }

func (m *Mock) Chat(ctx context.Context, req model.ChatRequest) (model.ChatResponse, error) {
	n := m.calls.Add(1)
	if m.cfg.Latency > 0 {
		t := time.NewTimer(m.cfg.Latency)
		defer t.Stop()
		select { case <-ctx.Done(): return model.ChatResponse{}, ctx.Err(); case <-t.C: }
	}
	if n <= m.cfg.FailFirst || (m.cfg.FailEvery > 0 && n%m.cfg.FailEvery == 0) {
		return model.ChatResponse{}, fmt.Errorf("%s injected failure: %w", m.cfg.Name, ErrUnavailable)
	}
	prompt := ""
	if len(req.Messages) > 0 { prompt = req.Messages[len(req.Messages)-1].Content }
	content := "mock response"
	finish := "stop"
	var calls []model.ToolCall
	if strings.Contains(prompt, "CALL:") && len(req.Tools) > 0 {
		name, args := parseDirective(prompt)
		if stablePercent(prompt) >= m.cfg.ToolQuality { name = "incorrect_tool" }
		argBytes, _ := json.Marshal(args)
		calls = []model.ToolCall{{ID: "call_mock", Type: "function", Function: model.FunctionCall{Name: name, Arguments: string(argBytes)}}}
		content, finish = "", "tool_calls"
	}
	return response(req.Model, content, calls, finish), nil
}

func parseDirective(prompt string) (string, map[string]any) {
	idx := strings.Index(prompt, "CALL:")
	part := strings.TrimSpace(prompt[idx+5:])
	fields := strings.SplitN(part, " ", 2)
	args := map[string]any{}
	if len(fields) == 2 { _ = json.Unmarshal([]byte(fields[1]), &args) }
	return fields[0], args
}

func stablePercent(s string) int {
	h := fnv.New32a(); _, _ = h.Write([]byte(s)); return int(h.Sum32() % 100)
}

func response(modelName, content string, calls []model.ToolCall, finish string) model.ChatResponse {
	c := content
	return model.ChatResponse{
		ID: "chatcmpl_mock", Object: "chat.completion", Created: time.Now().Unix(), Model: modelName,
		Choices: []model.Choice{{Index: 0, Message: model.AssistantMessage{Role: "assistant", Content: &c, ToolCalls: calls}, FinishReason: finish}},
	}
}
