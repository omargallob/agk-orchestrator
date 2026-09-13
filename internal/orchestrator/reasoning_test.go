package orchestrator

import (
	"testing"

	"github.com/omargallob/agk-orchestrator/internal/config"
)

func TestReasoningToolsConfigDisabled(t *testing.T) {
	// Reasoning off (the zero value) keeps the agent on the fast path: nil config.
	got := reasoningToolsConfig(config.AgentConfig{Name: "a"})
	if got != nil {
		t.Fatalf("reasoningToolsConfig with reasoning disabled = %+v, want nil", got)
	}
}

func TestReasoningToolsConfigEnabled(t *testing.T) {
	got := reasoningToolsConfig(config.AgentConfig{
		Name: "a",
		Tools: config.ToolConfig{
			Reasoning: config.ReasoningConfig{Enabled: true, MaxIterations: 3, MaxConcurrent: 2},
		},
	})
	if got == nil {
		t.Fatal("reasoningToolsConfig with reasoning enabled = nil, want non-nil")
	}
	if !got.Enabled {
		t.Error("v1beta ToolsConfig.Enabled = false, want true")
	}
	if got.MaxConcurrent != 2 {
		t.Errorf("MaxConcurrent = %d, want 2", got.MaxConcurrent)
	}
	if got.Reasoning == nil || !got.Reasoning.Enabled {
		t.Fatal("Reasoning not enabled")
	}
	if got.Reasoning.MaxIterations != 3 {
		t.Errorf("MaxIterations = %d, want 3", got.Reasoning.MaxIterations)
	}
}

func TestReasoningToolsConfigDefaultsMaxIterations(t *testing.T) {
	// Reasoning enabled but max_iterations unset falls back to the default.
	got := reasoningToolsConfig(config.AgentConfig{
		Name:  "a",
		Tools: config.ToolConfig{Reasoning: config.ReasoningConfig{Enabled: true}},
	})
	if got == nil || got.Reasoning == nil {
		t.Fatal("reasoningToolsConfig returned nil reasoning config")
	}
	if got.Reasoning.MaxIterations != defaultMaxIterations {
		t.Errorf("MaxIterations = %d, want default %d", got.Reasoning.MaxIterations, defaultMaxIterations)
	}
}

func TestBuildAgentWithReasoning(t *testing.T) {
	// An agent that opts into reasoning still builds successfully.
	agent, err := BuildAgent(config.AgentConfig{
		Name: "reasoner", Provider: "openai", Model: "m", BaseURL: "http://localhost:8080/v1",
		Tools: config.ToolConfig{Reasoning: config.ReasoningConfig{Enabled: true, MaxIterations: 4}},
	})
	if err != nil {
		t.Fatalf("BuildAgent: %v", err)
	}
	if agent == nil {
		t.Fatal("BuildAgent returned nil agent")
	}
}
