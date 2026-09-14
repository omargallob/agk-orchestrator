package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/omargallob/agk-orchestrator/internal/config"
)

func TestFromConfigRegistersEnabled(t *testing.T) {
	r, err := FromConfig(config.ToolsConfig{Enabled: []string{"fetch", "file", "shell"}, Allowlist: []string{"echo"}})
	if err != nil {
		t.Fatalf("FromConfig: %v", err)
	}
	got := strings.Join(r.Names(), ",")
	if got != "fetch,file,shell" {
		t.Errorf("Names() = %q", got)
	}
	if _, ok := r.Get("fetch"); !ok {
		t.Error("fetch not registered")
	}
}

func TestFetchTool(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("hello body"))
	}))
	defer srv.Close()

	ft := &fetchTool{client: srv.Client()}
	out, err := ft.Execute(context.Background(), map[string]string{"url": srv.URL})
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if out != "hello body" {
		t.Errorf("fetch = %q", out)
	}
	if _, err := ft.Execute(context.Background(), map[string]string{}); err == nil {
		t.Error("fetch with no url: want error")
	}
}

func TestFileTool(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("file contents"), 0o644); err != nil {
		t.Fatal(err)
	}
	ft := &fileTool{root: dir}
	out, err := ft.Execute(context.Background(), map[string]string{"path": "f.txt"})
	if err != nil {
		t.Fatalf("file: %v", err)
	}
	if out != "file contents" {
		t.Errorf("file = %q", out)
	}
	if _, err := ft.Execute(context.Background(), map[string]string{"path": "nope"}); err == nil {
		t.Error("missing file: want error")
	}
}

func TestFileToolSandbox(t *testing.T) {
	dir := t.TempDir()
	// A secret sibling outside the repo root must be unreachable.
	secret := filepath.Join(filepath.Dir(dir), "secret.txt")
	if err := os.WriteFile(secret, []byte("top secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(secret) }()

	ft := &fileTool{root: dir}
	for _, esc := range []string{"../secret.txt", secret, "sub/../../secret.txt"} {
		if _, err := ft.Execute(context.Background(), map[string]string{"path": esc}); err == nil {
			t.Errorf("path %q escaped the root but was allowed", esc)
		}
	}
}

func TestListTool(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "a.go"), "package a")
	mustWrite(t, filepath.Join(dir, "sub", "b.go"), "package b")
	mustWrite(t, filepath.Join(dir, ".git", "config"), "[core]")

	out, err := (&listTool{root: dir}).Execute(context.Background(), map[string]string{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	got := strings.Split(strings.TrimSpace(out), "\n")
	want := []string{"a.go", "sub/b.go"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("list = %v, want %v (must skip .git)", got, want)
	}
}

func TestGrepTool(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "a.go"), "package a\nfunc Target() {}\n")
	mustWrite(t, filepath.Join(dir, "b.go"), "package b\n")

	out, err := (&grepTool{root: dir}).Execute(context.Background(), map[string]string{"pattern": "func Target"})
	if err != nil {
		t.Fatalf("grep: %v", err)
	}
	if !strings.Contains(out, "a.go:2: func Target() {}") {
		t.Errorf("grep = %q, want a.go:2 match", out)
	}
	if _, err := (&grepTool{root: dir}).Execute(context.Background(), map[string]string{}); err == nil {
		t.Error("grep with no pattern: want error")
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestShellToolAllowlist(t *testing.T) {
	st := &shellTool{allowlist: map[string]bool{"echo": true}}

	out, err := st.Execute(context.Background(), map[string]string{"command": "echo hello there"})
	if err != nil {
		t.Fatalf("shell echo: %v", err)
	}
	if strings.TrimSpace(out) != "hello there" {
		t.Errorf("shell = %q", out)
	}

	// Not on the allowlist -> denied.
	if _, err := st.Execute(context.Background(), map[string]string{"command": "rm -rf /"}); err == nil || !strings.Contains(err.Error(), "allowlist") {
		t.Fatalf("expected allowlist denial, got %v", err)
	}
}

func TestUnknownToolErrors(t *testing.T) {
	if _, err := FromConfig(config.ToolsConfig{Enabled: []string{"telepathy"}}); err == nil {
		t.Error("unknown tool: want error")
	}
}
