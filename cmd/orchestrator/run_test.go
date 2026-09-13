package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveWorkflow(t *testing.T) {
	if w, err := resolveWorkflow([]string{"only"}, ""); err != nil || w != "only" {
		t.Errorf("single default: got %q, %v", w, err)
	}
	if w, err := resolveWorkflow([]string{"a", "b"}, "b"); err != nil || w != "b" {
		t.Errorf("explicit: got %q, %v", w, err)
	}
	if _, err := resolveWorkflow([]string{"a", "b"}, "c"); err == nil {
		t.Error("unknown requested: want error")
	}
	if _, err := resolveWorkflow([]string{"a", "b"}, ""); err == nil {
		t.Error("ambiguous: want error")
	}
	if _, err := resolveWorkflow(nil, ""); err == nil {
		t.Error("none: want error")
	}
}

func writeConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "wf.toml")
	body := `
[[agents]]
name = "a"
model = "m"
base_url = "http://localhost:8080/v1"

[[workflows]]
name = "main"
steps = ["a"]
`
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestResolveRunMissingConfig(t *testing.T) {
	if _, _, _, err := resolveRun("", "", nil, strings.NewReader("")); err == nil || !strings.Contains(err.Error(), "--config") {
		t.Fatalf("want --config error, got %v", err)
	}
}

func TestResolveRunFromArg(t *testing.T) {
	o, wf, input, err := resolveRun(writeConfig(t), "", []string{"hello", "world"}, strings.NewReader(""))
	if err != nil {
		t.Fatalf("resolveRun: %v", err)
	}
	if wf != "main" || input != "hello world" || o == nil {
		t.Errorf("got wf=%q input=%q o=%v", wf, input, o)
	}
}

func TestResolveRunFromStdin(t *testing.T) {
	_, wf, input, err := resolveRun(writeConfig(t), "main", nil, strings.NewReader("from stdin\n"))
	if err != nil {
		t.Fatalf("resolveRun: %v", err)
	}
	if wf != "main" || input != "from stdin" {
		t.Errorf("got wf=%q input=%q", wf, input)
	}
}

func TestResolveRunEmptyInput(t *testing.T) {
	if _, _, _, err := resolveRun(writeConfig(t), "", nil, strings.NewReader("   ")); err == nil || !strings.Contains(err.Error(), "no input") {
		t.Fatalf("want no-input error, got %v", err)
	}
}
