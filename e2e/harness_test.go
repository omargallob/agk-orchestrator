//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/omargallob/agk-orchestrator/internal/config"
	"github.com/omargallob/agk-orchestrator/internal/orchestrator"
)

// e2eQuestion is the question the harness asks. It can only be answered by
// reading the fixture repo (pricing.py defines BASE_PRICE_CENTS = 1999).
const e2eQuestion = "What is the base price in cents?"

// wantAnswer is the assertable fact from the fixture repo.
const wantAnswer = "1999"

// liveBaseURLEnv gates the live path against a real mlx_lm.server. When unset,
// the live test skips; the fake path always runs (including in CI).
const liveBaseURLEnv = "MLX_E2E_BASE_URL"

// materializeSampleRepo copies the embedded fixture repo to a fresh temp dir,
// emulating a newly cloned repo. It returns the temp dir path.
func materializeSampleRepo(t *testing.T) string {
	t.Helper()
	dst := t.TempDir()
	err := fs.WalkDir(SampleRepo, SampleRepoDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(SampleRepoDir, p)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := fs.ReadFile(SampleRepo, p)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, b, 0o644)
	})
	if err != nil {
		t.Fatalf("materialize sample repo: %v", err)
	}
	return dst
}

// scriptedFakeOpenAI scripts the deterministic tool-call dance the harness
// asserts:
//
//	call 0 -> TOOL_CALL grep for BASE_PRICE_CENTS
//	call 1 -> TOOL_CALL file read of pricing.py
//	call 2+ -> final answer containing 1999
//
// The continuation prompts built by the reasoning loop embed the previous tool
// results, so the bodies received on calls 1 and 2 prove the tools actually
// executed against the materialized repo. Bodies are recorded for assertions.
func scriptedFakeOpenAI(t *testing.T) (*httptest.Server, *int64, *[]string) {
	t.Helper()
	var calls int64
	var mu sync.Mutex
	bodies := []string{}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{"id": "test-model"}}})
			return
		}
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		bodies = append(bodies, string(body))
		mu.Unlock()
		n := atomic.AddInt64(&calls, 1)
		var reply string
		switch n {
		case 1:
			reply = `I will search the repo for the price. TOOL_CALL{"name":"grep","args":{"pattern":"BASE_PRICE_CENTS"}}`
		case 2:
			reply = `Found it in pricing.py. TOOL_CALL{"name":"file","args":{"path":"pricing.py"}}`
		default:
			reply = "The base price is 1999 cents."
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     "chatcmpl-e2e",
			"object": "chat.completion",
			"model":  "test-model",
			"choices": []map[string]any{{
				"index":         0,
				"message":       map[string]string{"role": "assistant", "content": reply},
				"finish_reason": "stop",
			}},
			"usage": map[string]int{"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2},
		})
	})), &calls, &bodies
}

// e2eConfig builds the orchestrator config from the embedded TOML, pointed at
// the given repo root and base URL.
func e2eConfig(t *testing.T, repoRoot, baseURL string) *config.Config {
	t.Helper()
	cfg, err := config.Parse([]byte(ConfigTOML))
	if err != nil {
		t.Fatalf("parse e2e config: %v", err)
	}
	cfg.Tools.Root = repoRoot
	for i := range cfg.Agents {
		cfg.Agents[i].BaseURL = baseURL
	}
	return cfg
}

func TestE2EFakeToolCallingSuite(t *testing.T) {
	repoRoot := materializeSampleRepo(t)
	// Sanity: the clone carries the asserted fact.
	if b, err := os.ReadFile(filepath.Join(repoRoot, "pricing.py")); err != nil || !strings.Contains(string(b), "BASE_PRICE_CENTS = "+wantAnswer) {
		t.Fatalf("materialized repo missing asserted fact BASE_PRICE_CENTS = %s", wantAnswer)
	}

	srv, _, bodies := scriptedFakeOpenAI(t)
	defer srv.Close()

	cfg := e2eConfig(t, repoRoot, srv.URL+"/v1")
	o, err := orchestrator.New(cfg)
	if err != nil {
		t.Fatalf("orchestrator.New: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	result, err := o.Run(ctx, "ask", e2eQuestion)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !result.Success {
		t.Fatalf("workflow failed: %s", result.Error)
	}
	if !strings.Contains(result.FinalOutput, wantAnswer) {
		t.Errorf("FinalOutput = %q, want it to contain %q", result.FinalOutput, wantAnswer)
	}
	// The scripted dance is grep -> file -> answer, so at least 3 LLM calls.
	if len(*bodies) < 3 {
		t.Fatalf("fake server saw %d chat/completions calls, want >= 3 (grep, file, answer)", len(*bodies))
	}
	// Call 2's continuation prompt embeds the grep result: proof grep ran
	// against the clone and found pricing.py.
	if !strings.Contains((*bodies)[1], "pricing.py") {
		t.Errorf("2nd LLM request does not mention pricing.py (grep result missing); body:\n%s", (*bodies)[1])
	}
	// Call 3's continuation prompt embeds the file contents: proof the agent
	// read pricing.py from the clone.
	if !strings.Contains((*bodies)[2], "BASE_PRICE_CENTS = "+wantAnswer) {
		t.Errorf("3rd LLM request does not contain the file contents (file result missing); body:\n%s", (*bodies)[2])
	}
}

func TestE2ELiveMLX(t *testing.T) {
	baseURL := os.Getenv(liveBaseURLEnv)
	if baseURL == "" {
		t.Skipf("live e2e skipped: %s is not set", liveBaseURLEnv)
	}
	repoRoot := materializeSampleRepo(t)
	cfg := e2eConfig(t, repoRoot, baseURL)
	o, err := orchestrator.New(cfg)
	if err != nil {
		t.Fatalf("orchestrator.New: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	result, err := o.Run(ctx, "ask", e2eQuestion)
	if err != nil {
		t.Fatalf("live Run: %v", err)
	}
	if !result.Success {
		t.Fatalf("live workflow failed: %s", result.Error)
	}
	// Looser assertions live: model phrasing varies; the fact must survive.
	if !strings.Contains(result.FinalOutput, wantAnswer) {
		t.Errorf("live FinalOutput = %q, want it to contain %q", result.FinalOutput, wantAnswer)
	}
}
