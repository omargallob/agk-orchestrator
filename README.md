# agk-orchestrator

A deployable multi-agent orchestrator built on [AgenticGoKit](https://github.com/agenticgokit/agenticgokit),
using a local **MLX** model (`mlx_lm.server`) as the reasoning brain. It runs as a
CLI (`run`) and a long-running HTTP service (`serve`), and ships as a multi-arch
container for Apple-Silicon Macs and the Raspberry Pi.

```
agk-orchestrator  →  AgenticGoKit (v1beta)  →  mlx_lm.server (OpenAI-compatible /v1)
   (this app)          (workflow engine)          (the model, via base_url)
```

MLX is reached through AgenticGoKit's built-in OpenAI-compatible adapter
(`provider = "openai"` or `"vllm"`) pointed at the server with `base_url` — no
custom provider code.

## Prerequisites

- An `mlx_lm.server` reachable over HTTP (OpenAI-compatible `/v1`). On the Mac:
  ```sh
  mlx_lm.server --model mlx-community/Qwen3-4B-Instruct-2507-8bit --host 0.0.0.0 --port 8080
  ```
- [`bazelisk`](https://github.com/bazelbuild/bazelisk) (builds/tests everything; a
  hermetic Go SDK is fetched by Bazel — no host Go needed except for golangci-lint).

## Configuration (TOML)

```toml
[[agents]]
name           = "researcher"
provider       = "openai"                 # or "vllm"; default "openai"
model          = "mlx-community/Qwen3-4B-Instruct-2507-8bit"
base_url       = "http://localhost:8080/v1"
system         = "You research things."   # static system prompt
# ...or a template instead of `system` (not both):
# prompt_template = "You are {agent} using {model}. Focus: {focus}."
# vars            = { focus = "biology" }

[[agents]]
name     = "writer"
model    = "mlx-community/Mistral-7B-Instruct-v0.3-4bit"
base_url = "http://pi.local:8080/v1"       # agents can target different hosts

[[workflows]]
name  = "main"
type  = "sequential"                       # "sequential" (default) or "parallel"
steps = ["researcher", "writer"]           # agent names

[tools]
enabled   = ["fetch", "file", "shell"]
allowlist = ["echo", "ls"]                 # programs the shell tool may run
```

- **Prompt templates** resolve `{var}` at build time: built-ins `{agent}`,
  `{model}`, `{provider}`, `{base_url}`, plus your `vars`. Unmatched placeholders
  are left as-is. Set `system` **or** `prompt_template`, not both.
- **Per-agent `base_url`** lets different agents use different `mlx_lm.server` hosts.

## CLI (`run`)

```sh
bazel run //cmd/orchestrator -- run --config workflow.toml "your input here"
# or pipe input:
echo "summarize this" | bazel run //cmd/orchestrator -- run --config workflow.toml
```

`--workflow <name>` selects a workflow (defaults to the only one if a single
workflow is defined). Input comes from the argument or stdin.

## HTTP service (`serve`)

```sh
bazel run //cmd/orchestrator -- serve --config workflow.toml --addr :8090
```

Async job API:

| Method & path | Purpose |
|---|---|
| `POST /jobs` | Submit `{"workflow":"main","input":"..."}` → `202` + `{id, status}` (`workflow` optional if only one). |
| `GET /jobs/{id}` | Job status/result: `queued`→`running`→`succeeded`/`failed`. |
| `GET /healthz` | Liveness. |

```sh
curl -s -XPOST localhost:8090/jobs -d '{"input":"say hi in three words"}'
curl -s localhost:8090/jobs/<id>
```

## Tools

The orchestrator executes tools itself (the model only emits tool-call requests):

- **fetch** — HTTP GET a URL (1 MiB cap).
- **file** — read a text file.
- **shell** — run an **allowlisted** program only; the command is whitespace-split
  with no shell interpretation (no pipes/redirects/chaining).

> Note: wiring these into the model's reasoning loop (a deterministic ReAct step
> using AgenticGoKit's `FormatToolsPromptForLLM` / `ParseLLMToolCalls`) is a
> planned follow-up; the tools and registry are in place.

## Deploy on the Raspberry Pi (docker compose)

```sh
cd deploy
cp orchestrator.example.toml orchestrator.toml   # edit base_url to the Mac's LAN IP
docker compose up -d
docker compose logs -f
```

The published image (`ghcr.io/omargallob/agk-orchestrator`) is multi-arch, so
`docker compose pull` fetches the arm64 build on the Pi. Its entrypoint is
`orchestrator serve`; the compose file supplies `--config` and `--addr`.

## Development

```sh
bazel test //...                              # unit + integration tests (nogo/vet inside the build)
bazel run //tools/lint:golangci-lint -- run   # lint
bazel run //:buildifier                       # format BUILD/.bzl/MODULE files
bazel run //:gazelle                          # regenerate BUILD files after adding/removing Go files
```

CI runs the tests + lint on an os/arch matrix (linux amd64/arm64, macOS arm64).
Releases are automated with Release Please (Conventional Commits → `CHANGELOG.md`
+ `vX.Y.Z` tags + versioned images).

## Out of scope (v1)

DAG/loop workflow topologies, run persistence, and HTTP auth (trusted LAN).
