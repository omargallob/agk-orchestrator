package config

import (
	"strings"
	"testing"
)

const validTOML = `
[[agents]]
name = "researcher"
model = "mlx-community/Qwen3-4B-Instruct-2507-8bit"
base_url = "http://localhost:8080/v1"
system = "You research things."

[[agents]]
name = "writer"
provider = "vllm"
model = "mlx-community/Mistral-7B-Instruct-v0.3-4bit"
base_url = "http://pi.local:8080/v1"

[[workflows]]
name = "main"
type = "sequential"
steps = ["researcher", "writer"]

[tools]
enabled = ["fetch", "shell"]
allowlist = ["echo", "ls"]
`

func TestParseValid(t *testing.T) {
	c, err := Parse([]byte(validTOML))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(c.Agents) != 2 || len(c.Workflows) != 1 {
		t.Fatalf("unexpected counts: %d agents, %d workflows", len(c.Agents), len(c.Workflows))
	}
	// Default provider applied.
	if c.Agents[0].Provider != "openai" {
		t.Errorf("agent[0].Provider = %q, want openai (default)", c.Agents[0].Provider)
	}
	if c.Agents[1].Provider != "vllm" {
		t.Errorf("agent[1].Provider = %q, want vllm", c.Agents[1].Provider)
	}
}

func TestParseErrors(t *testing.T) {
	cases := map[string]struct {
		toml string
		want string
	}{
		"malformed":         {"[[agents]] name = ", "parse config"},
		"no agents":         {`[tools]` + "\nenabled = []", "at least one"},
		"missing model":     {"[[agents]]\nname=\"a\"\nbase_url=\"http://x/v1\"", "model is required"},
		"missing base_url":  {"[[agents]]\nname=\"a\"\nmodel=\"m\"", "base_url is required"},
		"bad provider":      {"[[agents]]\nname=\"a\"\nprovider=\"anthropic\"\nmodel=\"m\"\nbase_url=\"http://x/v1\"", "unsupported provider"},
		"dup agent":         {"[[agents]]\nname=\"a\"\nmodel=\"m\"\nbase_url=\"http://x/v1\"\n[[agents]]\nname=\"a\"\nmodel=\"m\"\nbase_url=\"http://x/v1\"", "duplicate agent"},
		"bad wf type":       {"[[agents]]\nname=\"a\"\nmodel=\"m\"\nbase_url=\"http://x/v1\"\n[[workflows]]\nname=\"w\"\ntype=\"dag\"\nsteps=[\"a\"]", "unsupported type"},
		"wf unknown step":   {"[[agents]]\nname=\"a\"\nmodel=\"m\"\nbase_url=\"http://x/v1\"\n[[workflows]]\nname=\"w\"\nsteps=[\"nope\"]", "unknown agent"},
		"wf no steps":       {"[[agents]]\nname=\"a\"\nmodel=\"m\"\nbase_url=\"http://x/v1\"\n[[workflows]]\nname=\"w\"\nsteps=[]", "at least one step"},
		"unknown tool":      {"[[agents]]\nname=\"a\"\nmodel=\"m\"\nbase_url=\"http://x/v1\"\n[tools]\nenabled=[\"rm\"]", "unknown tool"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := Parse([]byte(tc.toml))
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %q, want it to contain %q", err, tc.want)
			}
		})
	}
}

func TestWorkflowDefaultsToSequential(t *testing.T) {
	c, err := Parse([]byte("[[agents]]\nname=\"a\"\nmodel=\"m\"\nbase_url=\"http://x/v1\"\n[[workflows]]\nname=\"w\"\nsteps=[\"a\"]"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if c.Workflows[0].Type != "sequential" {
		t.Errorf("workflow type = %q, want sequential (default)", c.Workflows[0].Type)
	}
}
