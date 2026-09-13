// Package tools provides the orchestrator's built-in tools (fetch, file, list,
// grep, shell) and a registry built from config. The filesystem tools (file,
// list, grep) and shell are confined to a configured repo root so an agent
// cannot read or run anything outside the repo it was given as context.
package tools

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/omargallob/agk-orchestrator/internal/config"
)

// maxListEntries and maxGrepMatches bound the filesystem tools' output so a
// large repo cannot blow up the model's context or the response size.
const (
	maxFileBytes   = 1 << 20 // 1 MiB
	maxListEntries = 2000
	maxGrepMatches = 200
)

// resolveInRoot resolves rel against root and rejects paths that escape it.
// An empty root defaults to the current directory. Absolute rel values are
// re-rooted (their leading separator is treated as relative to root), so a
// tool call cannot reach outside the repo.
func resolveInRoot(root, rel string) (string, error) {
	if root == "" {
		root = "."
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	target := filepath.Join(absRoot, rel)
	if target != absRoot && !strings.HasPrefix(target, absRoot+string(os.PathSeparator)) {
		return "", fmt.Errorf("path %q escapes the repo root", rel)
	}
	return target, nil
}

// Tool is a single executable capability the orchestrator can run on the model's
// behalf.
type Tool interface {
	Name() string
	Description() string
	// Execute runs the tool with string arguments and returns its textual result.
	Execute(ctx context.Context, args map[string]string) (string, error)
	// Parameters describes the tool's arguments so callers can build a schema
	// for native tool calling.
	Parameters() []Param
}

// Param describes a single tool argument.
type Param struct {
	Name        string
	Description string
	Required    bool
}

// Registry holds the enabled tools by name.
type Registry struct {
	tools map[string]Tool
}

// FromConfig builds a Registry from the enabled tools in cfg. The filesystem
// tools (file, list, grep) and shell are confined to cfg.Root; the shell tool is
// additionally gated by cfg.Allowlist (only listed programs may run).
func FromConfig(cfg config.ToolsConfig) (*Registry, error) {
	root := cfg.Root
	r := &Registry{tools: make(map[string]Tool)}
	for _, name := range cfg.Enabled {
		switch name {
		case "fetch":
			r.register(&fetchTool{client: &http.Client{Timeout: 30 * time.Second}})
		case "file":
			r.register(&fileTool{root: root})
		case "list":
			r.register(&listTool{root: root})
		case "grep":
			r.register(&grepTool{root: root})
		case "shell":
			r.register(&shellTool{allowlist: toSet(cfg.Allowlist), root: root})
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
func (fetchTool) Parameters() []Param {
	return []Param{{Name: "url", Description: "The URL to GET.", Required: true}}
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

// --- file (read-only, root-scoped) ---

type fileTool struct{ root string }

func (fileTool) Name() string { return "file" }
func (fileTool) Description() string {
	return "Read a text file within the repo and return its contents. Args: path (relative to the repo root)."
}
func (fileTool) Parameters() []Param {
	return []Param{{Name: "path", Description: "Path to the file, relative to the repo root.", Required: true}}
}

func (t *fileTool) Execute(_ context.Context, args map[string]string) (string, error) {
	rel := args["path"]
	if rel == "" {
		return "", fmt.Errorf("file: path is required")
	}
	path, err := resolveInRoot(t.root, rel)
	if err != nil {
		return "", fmt.Errorf("file: %w", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if len(b) > maxFileBytes {
		b = b[:maxFileBytes]
	}
	return string(b), nil
}

// --- list (read-only, root-scoped) ---

type listTool struct{ root string }

func (listTool) Name() string { return "list" }
func (listTool) Description() string {
	return "List files under a directory in the repo, recursively. Args: path (relative to the repo root; defaults to the root). Skips the .git directory."
}
func (listTool) Parameters() []Param {
	return []Param{{Name: "path", Description: "Directory to list, relative to the repo root. Defaults to the root.", Required: false}}
}

func (t *listTool) Execute(_ context.Context, args map[string]string) (string, error) {
	base, err := resolveInRoot(t.root, args["path"])
	if err != nil {
		return "", fmt.Errorf("list: %w", err)
	}
	root, err := resolveInRoot(t.root, "")
	if err != nil {
		return "", fmt.Errorf("list: %w", err)
	}
	var entries []string
	walkErr := filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		entries = append(entries, filepath.ToSlash(rel))
		if len(entries) >= maxListEntries {
			return filepath.SkipAll
		}
		return nil
	})
	if walkErr != nil {
		return "", fmt.Errorf("list: %w", walkErr)
	}
	sort.Strings(entries)
	return strings.Join(entries, "\n"), nil
}

// --- grep (read-only, root-scoped) ---

type grepTool struct{ root string }

func (grepTool) Name() string { return "grep" }
func (grepTool) Description() string {
	return "Search files in the repo for a regular expression and return matching lines as path:line: text. Args: pattern (required); path (relative directory to search, defaults to the root)."
}
func (grepTool) Parameters() []Param {
	return []Param{
		{Name: "pattern", Description: "Regular expression to search for.", Required: true},
		{Name: "path", Description: "Directory to search, relative to the repo root. Defaults to the root.", Required: false},
	}
}

func (t *grepTool) Execute(_ context.Context, args map[string]string) (string, error) {
	pattern := args["pattern"]
	if pattern == "" {
		return "", fmt.Errorf("grep: pattern is required")
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("grep: invalid pattern: %w", err)
	}
	base, err := resolveInRoot(t.root, args["path"])
	if err != nil {
		return "", fmt.Errorf("grep: %w", err)
	}
	root, err := resolveInRoot(t.root, "")
	if err != nil {
		return "", fmt.Errorf("grep: %w", err)
	}

	var matches []string
	walkErr := filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		relSlash := filepath.ToSlash(rel)
		f, err := os.Open(p)
		if err != nil {
			return nil // skip unreadable files rather than aborting the whole search
		}
		defer func() { _ = f.Close() }()
		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
		line := 0
		for scanner.Scan() {
			line++
			if re.MatchString(scanner.Text()) {
				matches = append(matches, fmt.Sprintf("%s:%d: %s", relSlash, line, strings.TrimSpace(scanner.Text())))
				if len(matches) >= maxGrepMatches {
					return filepath.SkipAll
				}
			}
		}
		return nil
	})
	if walkErr != nil {
		return "", fmt.Errorf("grep: %w", walkErr)
	}
	return strings.Join(matches, "\n"), nil
}

// --- shell (allowlisted) ---

type shellTool struct {
	allowlist map[string]bool
	root      string
}

func (shellTool) Name() string { return "shell" }
func (shellTool) Description() string {
	return "Run an allowlisted shell command in the repo directory and return its output. Args: command."
}
func (shellTool) Parameters() []Param {
	return []Param{{Name: "command", Description: "The allowlisted command to run.", Required: true}}
}
func (shellTool) Parameters() []Param {
	return []Param{{Name: "command", Description: "The allowlisted command to run.", Required: true}}
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
	cmd := exec.CommandContext(ctx, prog, fields[1:]...)
	// Confine execution to the repo root so commands operate on the repo.
	if dir, err := resolveInRoot(t.root, ""); err == nil {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
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
