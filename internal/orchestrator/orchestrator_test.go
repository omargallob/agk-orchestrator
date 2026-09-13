package orchestrator

import (
	"context"
	"strings"
	"testing"

	"github.com/agenticgokit/agenticgokit/v1beta"

	"github.com/omargallob/agk-orchestrator/internal/config"
)

func sampleConfig() *config.Config {
	return &config.Config{
		Agents: []config.AgentConfig{
			{Name: "researcher", Provider: "openai", Model: "m1", BaseURL: "http://localhost:8080/v1", System: "research"},
			{Name: "writer", Provider: "openai", Model: "m2", BaseURL: "http://localhost:8080/v1"},
		},
		Workflows: []config.WorkflowConfig{
			{Name: "main", Type: "sequential", Steps: []string{"researcher", "writer"}},
		},
	}
}

func TestBuildAgent(t *testing.T) {
	agent, err := BuildAgent(config.AgentConfig{
		Name: "a", Provider: "openai", Model: "m", BaseURL: "http://localhost:8080/v1",
	})
	if err != nil {
		t.Fatalf("BuildAgent: %v", err)
	}
	if agent == nil {
		t.Fatal("BuildAgent returned nil agent")
	}
}

func TestNewBuildsWorkflows(t *testing.T) {
	o, err := New(sampleConfig())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	got := o.Workflows()
	if len(got) != 1 || got[0] != "main" {
		t.Errorf("Workflows() = %v, want [main]", got)
	}
}

func TestBuildWorkflowUnknownAgent(t *testing.T) {
	_, err := BuildWorkflow(
		config.WorkflowConfig{Name: "w", Type: "sequential", Steps: []string{"ghost"}},
		map[string]v1beta.Agent{},
	)
	if err == nil || !strings.Contains(err.Error(), "unknown agent") {
		t.Fatalf("expected unknown-agent error, got %v", err)
	}
}

func TestBuildWorkflowParallel(t *testing.T) {
	agents, err := BuildAgents(sampleConfig().Agents)
	if err != nil {
		t.Fatalf("BuildAgents: %v", err)
	}
	wf, err := BuildWorkflow(config.WorkflowConfig{Name: "p", Type: "parallel", Steps: []string{"researcher", "writer"}}, agents)
	if err != nil {
		t.Fatalf("BuildWorkflow parallel: %v", err)
	}
	if wf == nil {
		t.Fatal("nil workflow")
	}
}

func TestResolveSystemPrompt(t *testing.T) {
	// Static system prompt passes through.
	if got := resolveSystemPrompt(config.AgentConfig{System: "hello"}); got != "hello" {
		t.Errorf("static = %q", got)
	}
	// Template with built-in vars + user vars.
	a := config.AgentConfig{
		Name:           "researcher",
		Provider:       "openai",
		Model:          "qwen",
		BaseURL:        "http://x/v1",
		PromptTemplate: "You are {agent} using {model}. Focus: {focus}.",
		Vars:           map[string]string{"focus": "biology"},
	}
	want := "You are researcher using qwen. Focus: biology."
	if got := resolveSystemPrompt(a); got != want {
		t.Errorf("template = %q, want %q", got, want)
	}
}

func TestRunUnknownWorkflow(t *testing.T) {
	o, err := New(sampleConfig())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := o.Run(context.Background(), "nope", "hi"); err == nil {
		t.Fatal("expected error for unknown workflow")
	}
}
