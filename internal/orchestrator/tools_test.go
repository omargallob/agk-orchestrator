package orchestrator

import (
	"context"
	"errors"
	"testing"

	"github.com/agenticgokit/agenticgokit/v1beta"

	"github.com/omargallob/agk-orchestrator/internal/config"
	"github.com/omargallob/agk-orchestrator/internal/tools"
)

// stubTool is a minimal tools.Tool for exercising the adapter.
type stubTool struct {
	gotArgs map[string]string
	out     string
	err     error
}

func (s *stubTool) Name() string        { return "stub" }
func (s *stubTool) Description() string { return "a stub tool" }
func (s *stubTool) Parameters() []tools.Param {
	return []tools.Param{{Name: "x", Description: "an x", Required: true}}
}
func (s *stubTool) Execute(_ context.Context, args map[string]string) (string, error) {
	s.gotArgs = args
	return s.out, s.err
}

func TestToolAdapterExecuteSuccess(t *testing.T) {
	stub := &stubTool{out: "done"}
	a := &toolAdapter{tool: stub}

	res, err := a.Execute(context.Background(), map[string]interface{}{"x": 42})
	if err != nil {
		t.Fatalf("Execute returned Go error: %v", err)
	}
	if !res.Success {
		t.Errorf("Success = false, want true (Error=%q)", res.Error)
	}
	if res.Content != "done" {
		t.Errorf("Content = %v, want %q", res.Content, "done")
	}
	// Untyped args are stringified for the internal tool.
	if stub.gotArgs["x"] != "42" {
		t.Errorf("tool received x=%q, want %q", stub.gotArgs["x"], "42")
	}
}

func TestToolAdapterExecuteError(t *testing.T) {
	a := &toolAdapter{tool: &stubTool{err: errors.New("boom")}}

	res, err := a.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("Execute should report tool failure in-band, got Go error: %v", err)
	}
	if res.Success {
		t.Error("Success = true, want false on tool error")
	}
	if res.Error != "boom" {
		t.Errorf("Error = %q, want %q", res.Error, "boom")
	}
}

func TestToolAdapterJSONSchema(t *testing.T) {
	a := &toolAdapter{tool: &stubTool{}}
	schema := a.JSONSchema()

	if schema["type"] != "object" {
		t.Errorf("schema type = %v, want object", schema["type"])
	}
	props, ok := schema["properties"].(map[string]interface{})
	if !ok || props["x"] == nil {
		t.Fatalf("properties missing x: %v", schema["properties"])
	}
	req, ok := schema["required"].([]string)
	if !ok || len(req) != 1 || req[0] != "x" {
		t.Errorf("required = %v, want [x]", schema["required"])
	}
}

// toolAdapter must satisfy ToolWithSchema so native tool calling can advertise it.
var _ v1beta.ToolWithSchema = (*toolAdapter)(nil)

func TestRegisterToolsAreDiscoverable(t *testing.T) {
	if err := registerTools(config.ToolsConfig{Enabled: []string{"file", "fetch"}}); err != nil {
		t.Fatalf("registerTools: %v", err)
	}

	discovered, err := v1beta.DiscoverInternalTools()
	if err != nil {
		t.Fatalf("DiscoverInternalTools: %v", err)
	}
	names := map[string]bool{}
	for _, tl := range discovered {
		names[tl.Name()] = true
	}
	for _, want := range []string{"file", "fetch"} {
		if !names[want] {
			t.Errorf("tool %q not discoverable after registerTools; got %v", want, names)
		}
	}
}
