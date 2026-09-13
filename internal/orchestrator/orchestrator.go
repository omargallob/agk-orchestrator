// Package orchestrator builds AgenticGoKit v1beta agents and workflows from the
// orchestrator config and runs them. AgenticGoKit provides the workflow engine
// (sequential/parallel); this package is the config-driven wiring on top.
package orchestrator

import (
	"context"
	"fmt"
	"time"

	"github.com/agenticgokit/agenticgokit/v1beta"

	"github.com/omargallob/agk-orchestrator/internal/config"
	"github.com/omargallob/agk-orchestrator/internal/prompt"
)

// placeholderAPIKey satisfies AgenticGoKit's OpenAI-compatible adapter, which
// rejects an empty API key. mlx_lm.server does not check Authorization, so any
// non-empty value works.
const placeholderAPIKey = "mlx-local-no-auth"

// defaultAgentTimeout is used when an agent config sets no timeout; v1beta
// requires a positive agent timeout.
const defaultAgentTimeout = 5 * time.Minute

// defaultMaxIterations bounds the reasoning loop when an agent enables reasoning
// but sets no max_iterations. Matches AgenticGoKit's own default.
const defaultMaxIterations = 5

// Orchestrator holds the agents and workflows built from a Config.
type Orchestrator struct {
	agents    map[string]v1beta.Agent
	workflows map[string]v1beta.Workflow
}

// New builds all agents and workflows described by cfg.
func New(cfg *config.Config) (*Orchestrator, error) {
	// Expose enabled tools to AgenticGoKit before building agents so that
	// reasoning-enabled agents discover them.
	if err := registerTools(cfg.Tools); err != nil {
		return nil, err
	}
	agents, err := BuildAgents(cfg.Agents)
	if err != nil {
		return nil, err
	}
	workflows := make(map[string]v1beta.Workflow, len(cfg.Workflows))
	for _, w := range cfg.Workflows {
		wf, err := BuildWorkflow(w, agents)
		if err != nil {
			return nil, err
		}
		workflows[w.Name] = wf
	}
	return &Orchestrator{agents: agents, workflows: workflows}, nil
}

// Workflows returns the configured workflow names.
func (o *Orchestrator) Workflows() []string {
	names := make([]string, 0, len(o.workflows))
	for name := range o.workflows {
		names = append(names, name)
	}
	return names
}

// Run executes the named workflow with the given input.
func (o *Orchestrator) Run(ctx context.Context, workflow, input string) (*v1beta.WorkflowResult, error) {
	wf, ok := o.workflows[workflow]
	if !ok {
		return nil, fmt.Errorf("unknown workflow %q", workflow)
	}
	return wf.Run(ctx, input)
}

// BuildAgents builds every agent, keyed by name.
func BuildAgents(cfgs []config.AgentConfig) (map[string]v1beta.Agent, error) {
	agents := make(map[string]v1beta.Agent, len(cfgs))
	for _, a := range cfgs {
		agent, err := BuildAgent(a)
		if err != nil {
			return nil, err
		}
		agents[a.Name] = agent
	}
	return agents, nil
}

// BuildAgent constructs a v1beta agent pointed at the configured mlx_lm.server
// via base_url, using AgenticGoKit's OpenAI-compatible adapter.
func BuildAgent(a config.AgentConfig) (v1beta.Agent, error) {
	cfg := &v1beta.Config{
		Name:         a.Name,
		SystemPrompt: resolveSystemPrompt(a),
		Timeout:      defaultAgentTimeout,
		LLM: v1beta.LLMConfig{
			Provider: a.Provider,
			Model:    a.Model,
			BaseURL:  a.BaseURL,
			APIKey:   placeholderAPIKey,
		},
		// Memory/RAG is out of scope for v1; keep agents lightweight.
		Memory: &v1beta.MemoryConfig{Enabled: false},
		// Tools/reasoning: when the agent opts in, AgenticGoKit runs the
		// tool-calling continuation loop internally; nil keeps the fast path.
		Tools: reasoningToolsConfig(a),
	}
	agent, err := v1beta.NewBuilder(a.Name).WithConfig(cfg).Build()
	if err != nil {
		return nil, fmt.Errorf("build agent %q: %w", a.Name, err)
	}
	return agent, nil
}

// reasoningToolsConfig maps our per-agent reasoning config onto AgenticGoKit's
// v1beta ToolsConfig. It returns nil when reasoning is disabled so the agent
// stays on the single-call fast path.
func reasoningToolsConfig(a config.AgentConfig) *v1beta.ToolsConfig {
	r := a.Tools.Reasoning
	if !r.Enabled {
		return nil
	}
	maxIterations := r.MaxIterations
	if maxIterations <= 0 {
		maxIterations = defaultMaxIterations
	}
	return &v1beta.ToolsConfig{
		Enabled:       true,
		MaxConcurrent: r.MaxConcurrent,
		Reasoning: &v1beta.ReasoningConfig{
			Enabled:       true,
			MaxIterations: maxIterations,
		},
	}
}

// resolveSystemPrompt returns the agent's system prompt: the static System, or,
// if PromptTemplate is set, the template resolved with built-in vars ({agent},
// {model}, {provider}, {base_url}) plus the agent's Vars.
func resolveSystemPrompt(a config.AgentConfig) string {
	if a.PromptTemplate == "" {
		return a.System
	}
	vars := map[string]string{
		"agent":    a.Name,
		"model":    a.Model,
		"provider": a.Provider,
		"base_url": a.BaseURL,
	}
	for k, v := range a.Vars {
		vars[k] = v
	}
	return prompt.Resolve(a.PromptTemplate, vars)
}

// BuildWorkflow assembles a v1beta workflow from a WorkflowConfig and the built
// agents. Only sequential and parallel are supported in v1.
func BuildWorkflow(w config.WorkflowConfig, agents map[string]v1beta.Agent) (v1beta.Workflow, error) {
	var (
		wf  v1beta.Workflow
		err error
	)
	switch w.Type {
	case "sequential":
		wf, err = v1beta.NewSequentialWorkflow(nil)
	case "parallel":
		wf, err = v1beta.NewParallelWorkflow(nil)
	default:
		return nil, fmt.Errorf("workflow %q: unsupported type %q (want sequential or parallel)", w.Name, w.Type)
	}
	if err != nil {
		return nil, fmt.Errorf("workflow %q: %w", w.Name, err)
	}

	for _, step := range w.Steps {
		agent, ok := agents[step]
		if !ok {
			return nil, fmt.Errorf("workflow %q: unknown agent %q", w.Name, step)
		}
		if err := wf.AddStep(v1beta.WorkflowStep{Name: step, Agent: agent}); err != nil {
			return nil, fmt.Errorf("workflow %q: add step %q: %w", w.Name, step, err)
		}
	}
	return wf, nil
}
