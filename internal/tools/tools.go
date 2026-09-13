// Package tools provides the orchestrator's built-in tools (fetch, file, shell)
// and a registry built from config. The deterministic orchestrator executes
// these tools itself (the model only emits tool-call requests).
package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/omargallob/agk-orchestrator/internal/config"
)

// Tool is a single executable capability the orchestrator can run on the model's
// behalf.
type Tool interface {
	Name() string
	Description() string
	// Execute runs the tool with string arguments and returns its textual result.
	Execute(ctx context.Context, args map[string]string) (string, error)
}

// Registry holds the enabled tools by name.
type Registry struct {
	tools map[string]Tool
}

// FromConfig builds a Registry from the enabled tools in cfg. The shell tool is
// gated by cfg.Allowlist (only listed programs may run).
func FromConfig(cfg config.ToolsConfig) (*Registry, error) {
	r := &Registry{tools: make(map[string]Tool)}
	for _, name := range cfg.Enabled {
		switch name {
		case "fetch":
			r.register(&fetchTool{client: &http.Client{Timeout: 30 * time.Second}})
		case "file":
			r.register(&fileTool{})
		case "shell":
			r.register(&shellTool{allowlist: toSet(cfg.Allowlist)})
		default:
			return nil, fmt.Errorf("tools: unknown tool %q", name)
		}
	}
	return r, nil
}

func (r *Registry) register(t Tool) { r.tools[t.Name()] = t }

// Get returns the named tool.
func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// Names returns the registered tool names, sorted.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.tools))
	for n := range r.tools {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// --- fetch ---

type fetchTool struct{ client *http.Client }

func (fetchTool) Name() string { return "fetch" }
func (fetchTool) Description() string {
	return "HTTP GET a URL and return the response body. Args: url."
}

func (t *fetchTool) Execute(ctx context.Context, args map[string]string) (string, error) {
	url := args["url"]
	if url == "" {
		return "", fmt.Errorf("fetch: url is required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // cap at 1 MiB
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return string(body), nil
}

// --- file (read-only) ---

type fileTool struct{}

func (fileTool) Name() string        { return "file" }
func (fileTool) Description() string { return "Read a text file and return its contents. Args: path." }

func (fileTool) Execute(_ context.Context, args map[string]string) (string, error) {
	path := args["path"]
	if path == "" {
		return "", fmt.Errorf("file: path is required")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// --- shell (allowlisted) ---

type shellTool struct{ allowlist map[string]bool }

func (shellTool) Name() string { return "shell" }
func (shellTool) Description() string {
	return "Run an allowlisted shell command and return its output. Args: command."
}

func (t *shellTool) Execute(ctx context.Context, args map[string]string) (string, error) {
	command := strings.TrimSpace(args["command"])
	if command == "" {
		return "", fmt.Errorf("shell: command is required")
	}
	// Split on whitespace (no shell interpretation) so only the named program
	// runs — no pipes, redirects, or chaining.
	fields := strings.Fields(command)
	prog := fields[0]
	if !t.allowlist[prog] {
		return "", fmt.Errorf("shell: %q is not in the allowlist", prog)
	}
	out, err := exec.CommandContext(ctx, prog, fields[1:]...).CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("shell: %q: %w", command, err)
	}
	return string(out), nil
}

func toSet(items []string) map[string]bool {
	set := make(map[string]bool, len(items))
	for _, it := range items {
		set[it] = true
	}
	return set
}
