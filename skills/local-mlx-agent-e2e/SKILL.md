---
name: local-mlx-agent-e2e
description: Run a local MLX model (mlx_lm.server on Apple silicon) as the brain of a tool-calling agent, and write end-to-end tests for it in Go. Use when wiring an agent framework to a local model via an OpenAI-compatible endpoint, testing multi-step tool calling deterministically in CI, building a scripted fake OpenAI server with httptest, gating live-model tests behind an env var, or debugging flaky tool calls from small local models.
---

# Local MLX agents + end-to-end tests

## Use MLX through the OpenAI-compatible API

`mlx_lm.server` exposes an OpenAI-compatible `/v1/chat/completions`. Don't
write a custom provider; point your framework's OpenAI adapter at it.

```bash
pip install mlx-lm
mlx_lm.server --model mlx-community/Qwen3-4B-Instruct-2507-8bit --port 8080
```

Then configure the agent with `base_url = "http://localhost:8080/v1"` and the
model ID as `model`. Any API key value works.

**Model choice for tool calling:** Qwen and Llama instruct families are the
most consistent at emitting well-formed tool calls. Small models will
sometimes loop (call tools forever) or answer without calling tools. Always:

- cap the reasoning loop (`max_iterations` ~5),
- keep `max_concurrent` tool execution at 1 unless tools are independent,
- allowlist any shell tool to specific commands.

## Two-tier e2e harness

Real models are slow and non-deterministic; CI has no GPU. Split the suite:

| Tier | Backend | Runs | Asserts |
|---|---|---|---|
| Scripted | `httptest` fake OpenAI server | always, incl. CI | exact tool sequence + that tools really ran |
| Live | real `mlx_lm.server` | only when an env var is set | the final answer contains the fact |

Both tiers run the **same** agent config and question; only `base_url` differs.

### 1. A fixture repo with one assertable fact

Embed a tiny sample repo whose answer can *only* be found by reading code
(e.g. `pricing.py` contains `BASE_PRICE_CENTS = 1999`; ask "What is the base
price in cents?"). Materialize it into `t.TempDir()` per test so tools operate
on a fresh "clone":

```go
//go:embed sample-repo
var SampleRepo embed.FS

func materialize(t *testing.T) string {
	t.Helper()
	dst := t.TempDir()
	err := fs.WalkDir(SampleRepo, "sample-repo", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel("sample-repo", p)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := fs.ReadFile(SampleRepo, p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, 0o644)
	})
	if err != nil {
		t.Fatalf("materialize: %v", err)
	}
	return dst
}
```

Point the agent's file/grep tools at that directory as their root.

### 2. Scripted fake OpenAI server

Respond by call count to script the tool-call dance, and **record request
bodies**. The framework's continuation prompts embed previous tool results, so
the bodies prove the tools actually executed:

```go
func scriptedFake(t *testing.T) (*httptest.Server, *[]string) {
	t.Helper()
	var calls int64
	var mu sync.Mutex
	bodies := []string{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost { // model listing / health probes
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{"id": "test-model"}}})
			return
		}
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		bodies = append(bodies, string(body))
		mu.Unlock()
		var reply string
		switch atomic.AddInt64(&calls, 1) {
		case 1:
			reply = `<tool call: grep BASE_PRICE_CENTS>` // in your framework's tool-call format
		case 2:
			reply = `<tool call: read pricing.py>`
		default:
			reply = "The base price is 1999 cents."
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "chatcmpl-e2e", "object": "chat.completion", "model": "test-model",
			"choices": []map[string]any{{
				"index": 0, "finish_reason": "stop",
				"message": map[string]string{"role": "assistant", "content": reply},
			}},
			"usage": map[string]int{"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2},
		})
	}))
	t.Cleanup(srv.Close)
	return srv, &bodies
}
```

Use the exact tool-call syntax your framework parses (text markers or the
`tool_calls` field). Then assert:

- final output contains `1999`;
- `len(bodies) >= 3` (grep → read → answer);
- `bodies[1]` contains `pricing.py` (grep ran against the clone);
- `bodies[2]` contains `BASE_PRICE_CENTS = 1999` (the file was read).

The last two catch the most common silent failure: the model "answers" but the
tool never ran, or ran against the wrong root.

### 3. Live test, env-gated

```go
func TestE2ELiveMLX(t *testing.T) {
	baseURL := os.Getenv("MLX_E2E_BASE_URL")
	if baseURL == "" {
		t.Skip("set MLX_E2E_BASE_URL to run against a real mlx_lm.server")
	}
	// same config + question, base_url = baseURL, timeout ~5m
	// assert only: final output contains "1999" (phrasing varies)
}
```

Keep live assertions loose: check the fact, not the tool sequence.

### 4. Keep it out of the default test run

Put the harness behind a build tag (`//go:build e2e`) so `go test ./...` stays
fast. Under Bazel, set `gotags = ["e2e"]` and `tags = ["manual"]` on the
`go_test`, and run it explicitly:

```bash
go test -tags e2e ./e2e/...
MLX_E2E_BASE_URL=http://localhost:8080/v1 go test -tags e2e -run Live ./e2e/...
```

## Debugging flaky live runs

- Read the recorded request/response bodies first; most failures are a
  malformed tool call the parser silently dropped.
- Lower temperature; tool calling on small models degrades fast above ~0.3.
- If the model never calls tools, make the system prompt name the tools and
  say the answer must come from the repository.
- If it loops, confirm `max_iterations` is set and that tool results are
  actually fed back into the continuation prompt.
