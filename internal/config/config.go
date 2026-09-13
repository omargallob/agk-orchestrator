// Package config loads and validates the orchestrator's TOML configuration:
// agents, workflows, and tools.
package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// validProviders are AgenticGoKit's built-in OpenAI-compatible adapters that can
// be pointed at an mlx_lm.server via base_url.
var validProviders = map[string]bool{"openai": true, "vllm": true}

// validWorkflowTypes are the v1 workflow topologies.
var validWorkflowTypes = map[string]bool{"sequential": true, "parallel": true}

// validTools are the built-in orchestrator tools.
var validTools = map[string]bool{"fetch": true, "file": true, "shell": true}

// Config is the top-level orchestrator configuration.
type Config struct {
	Agents    []AgentConfig    `toml:"agents"`
	Workflows []WorkflowConfig `toml:"workflows"`
	Tools     ToolsConfig      `toml:"tools"`
}

// AgentConfig configures a single agent backed by an mlx_lm.server through
// AgenticGoKit's OpenAI-compatible adapter.
type AgentConfig struct {
	Name     string `toml:"name"`
	Provider string `toml:"provider"` // "openai" (default) or "vllm"
	Model    string `toml:"model"`
	BaseURL  string `toml:"base_url"` // mlx_lm.server /v1 endpoint
	System   string `toml:"system"`   // optional system prompt
}

// WorkflowConfig wires agents into a sequential or parallel workflow.
type WorkflowConfig struct {
	Name  string   `toml:"name"`
	Type  string   `toml:"type"`  // "sequential" (default) or "parallel"
	Steps []string `toml:"steps"` // agent names, in order
}

// ToolsConfig enables orchestrator tools and gates shell/exec via an allowlist.
type ToolsConfig struct {
	Enabled   []string `toml:"enabled"`
	Allowlist []string `toml:"allowlist"`
}

// Load reads and validates a TOML config file.
func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(b)
}

// Parse decodes and validates TOML config bytes.
func Parse(b []byte) (*Config, error) {
	var c Config
	if err := toml.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	c.applyDefaults()
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

// applyDefaults fills in optional fields before validation.
func (c *Config) applyDefaults() {
	for i := range c.Agents {
		if c.Agents[i].Provider == "" {
			c.Agents[i].Provider = "openai"
		}
	}
	for i := range c.Workflows {
		if c.Workflows[i].Type == "" {
			c.Workflows[i].Type = "sequential"
		}
	}
}

// Validate checks the config for structural and referential errors.
func (c *Config) Validate() error {
	if len(c.Agents) == 0 {
		return fmt.Errorf("config: at least one [[agents]] entry is required")
	}

	agents := make(map[string]bool, len(c.Agents))
	for i, a := range c.Agents {
		switch {
		case a.Name == "":
			return fmt.Errorf("agents[%d]: name is required", i)
		case agents[a.Name]:
			return fmt.Errorf("agents[%d]: duplicate agent name %q", i, a.Name)
		case !validProviders[a.Provider]:
			return fmt.Errorf("agent %q: unsupported provider %q (want openai or vllm)", a.Name, a.Provider)
		case a.Model == "":
			return fmt.Errorf("agent %q: model is required", a.Name)
		case a.BaseURL == "":
			return fmt.Errorf("agent %q: base_url is required (the mlx_lm.server /v1 endpoint)", a.Name)
		}
		agents[a.Name] = true
	}

	workflows := make(map[string]bool, len(c.Workflows))
	for i, w := range c.Workflows {
		switch {
		case w.Name == "":
			return fmt.Errorf("workflows[%d]: name is required", i)
		case workflows[w.Name]:
			return fmt.Errorf("workflows[%d]: duplicate workflow name %q", i, w.Name)
		case !validWorkflowTypes[w.Type]:
			return fmt.Errorf("workflow %q: unsupported type %q (want sequential or parallel)", w.Name, w.Type)
		case len(w.Steps) == 0:
			return fmt.Errorf("workflow %q: at least one step is required", w.Name)
		}
		for _, step := range w.Steps {
			if !agents[step] {
				return fmt.Errorf("workflow %q: step references unknown agent %q", w.Name, step)
			}
		}
		workflows[w.Name] = true
	}

	for _, t := range c.Tools.Enabled {
		if !validTools[t] {
			return fmt.Errorf("tools: unknown tool %q (want fetch, file, or shell)", t)
		}
	}
	return nil
}
