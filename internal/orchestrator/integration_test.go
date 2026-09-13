package orchestrator

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/omargallob/agk-orchestrator/internal/config"
)

// fakeOpenAI implements the minimal OpenAI-compatible surface AgenticGoKit's
// adapter uses, so we can exercise the full config -> agent -> workflow -> result
// chain without a live mlx_lm.server.
func fakeOpenAI(t *testing.T, reply string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost { // chat/completions
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":     "chatcmpl-test",
				"object": "chat.completion",
				"model":  "test-model",
				"choices": []map[string]any{{
					"index":         0,
					"message":       map[string]string{"role": "assistant", "content": reply},
					"finish_reason": "stop",
				}},
				"usage": map[string]int{"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{"id": "test-model"}}})
	}))
}

func TestIntegrationSequentialWorkflow(t *testing.T) {
	srv := fakeOpenAI(t, "INTEGRATION_OK")
	defer srv.Close()

	cfg := &config.Config{
		Agents: []config.AgentConfig{
			{Name: "assistant", Provider: "openai", Model: "test-model", BaseURL: srv.URL + "/v1", System: "be brief"},
		},
		Workflows: []config.WorkflowConfig{
			{Name: "main", Type: "sequential", Steps: []string{"assistant"}},
		},
	}

	o, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	result, err := o.Run(context.Background(), "main", "hello")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !result.Success {
		t.Fatalf("workflow failed: %s", result.Error)
	}
	if !strings.Contains(result.FinalOutput, "INTEGRATION_OK") {
		t.Errorf("FinalOutput = %q, want it to contain INTEGRATION_OK", result.FinalOutput)
	}
}

func TestIntegrationParallelWorkflow(t *testing.T) {
	srv := fakeOpenAI(t, "PARALLEL_OK")
	defer srv.Close()

	cfg := &config.Config{
		Agents: []config.AgentConfig{
			{Name: "a", Provider: "openai", Model: "test-model", BaseURL: srv.URL + "/v1"},
			{Name: "b", Provider: "openai", Model: "test-model", BaseURL: srv.URL + "/v1"},
		},
		Workflows: []config.WorkflowConfig{
			{Name: "fanout", Type: "parallel", Steps: []string{"a", "b"}},
		},
	}

	o, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	result, err := o.Run(context.Background(), "fanout", "hello")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !result.Success {
		t.Fatalf("workflow failed: %s", result.Error)
	}
	if !strings.Contains(result.FinalOutput, "PARALLEL_OK") {
		t.Errorf("FinalOutput = %q, want it to contain PARALLEL_OK", result.FinalOutput)
	}
}
