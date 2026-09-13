package orchestrator

import (
	"context"
	"fmt"

	"github.com/agenticgokit/agenticgokit/v1beta"

	"github.com/omargallob/agk-orchestrator/internal/config"
	"github.com/omargallob/agk-orchestrator/internal/tools"
)

// registerTools exposes the orchestrator's enabled tools to AgenticGoKit's
// internal tool registry. v1beta auto-discovers registered tools (via
// DiscoverInternalTools) and offers them to any reasoning-enabled agent, so
// this is the single wiring point that connects internal/tools to the agents.
func registerTools(cfg config.ToolsConfig) error {
	reg, err := tools.FromConfig(cfg)
	if err != nil {
		return err
	}
	for _, name := range reg.Names() {
		t, ok := reg.Get(name)
		if !ok {
			continue
		}
		tool := t // capture per iteration for the factory closure
		v1beta.RegisterInternalTool(name, func() v1beta.Tool {
			return &toolAdapter{tool: tool}
		})
	}
	return nil
}

// toolAdapter adapts an internal tools.Tool to AgenticGoKit's v1beta.Tool. It
// also implements ToolWithSchema so the OpenAI-compatible adapter can advertise
// each tool for native tool calling against mlx_lm.server.
type toolAdapter struct{ tool tools.Tool }

func (a *toolAdapter) Name() string        { return a.tool.Name() }
func (a *toolAdapter) Description() string { return a.tool.Description() }

// Execute converts v1beta's untyped args to the internal tool's string args and
// maps the result back. Tool failures are reported in-band via ToolResult so the
// model can see and react to them, matching AgenticGoKit's own tools.
func (a *toolAdapter) Execute(ctx context.Context, args map[string]interface{}) (*v1beta.ToolResult, error) {
	strArgs := make(map[string]string, len(args))
	for k, v := range args {
		strArgs[k] = fmt.Sprint(v)
	}
	out, err := a.tool.Execute(ctx, strArgs)
	if err != nil {
		return &v1beta.ToolResult{Success: false, Error: err.Error()}, nil
	}
	return &v1beta.ToolResult{Success: true, Content: out}, nil
}

// JSONSchema returns the tool's argument schema for native tool calling. All
// arguments are strings; tools with no parameters yield an empty object schema.
func (a *toolAdapter) JSONSchema() map[string]interface{} {
	properties := map[string]interface{}{}
	required := []string{}
	for _, p := range a.tool.Parameters() {
		properties[p.Name] = map[string]interface{}{
			"type":        "string",
			"description": p.Description,
		}
		if p.Required {
			required = append(required, p.Name)
		}
	}
	return map[string]interface{}{
		"type":       "object",
		"properties": properties,
		"required":   required,
	}
}
