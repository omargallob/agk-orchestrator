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
	p := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(p, []byte("file contents"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := fileTool{}.Execute(context.Background(), map[string]string{"path": p})
	if err != nil {
		t.Fatalf("file: %v", err)
	}
	if out != "file contents" {
		t.Errorf("file = %q", out)
	}
	if _, err := (fileTool{}).Execute(context.Background(), map[string]string{"path": filepath.Join(dir, "nope")}); err == nil {
		t.Error("missing file: want error")
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
