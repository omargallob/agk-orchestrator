# agk-orchestrator

[![codecov](https://codecov.io/gh/omargallob/agk-orchestrator/branch/main/graph/badge.svg)](https://codecov.io/gh/omargallob/agk-orchestrator)

Deployable multi-agent orchestrator on [AgenticGoKit](https://github.com/agenticgokit/agenticgokit)
using a local MLX model (`mlx_lm.server`) as the reasoning brain. MLX is reached
via AgenticGoKit's built-in OpenAI-compatible adapter (`provider = "openai"` or
`"vllm"`) pointed at the server with `base_url` — no custom provider.

## Prompt templates

An agent's system prompt can be a static `system` string, or a `prompt_template`
with `{var}` placeholders resolved at build time. Built-in variables: `{agent}`,
`{model}`, `{provider}`, `{base_url}`; add your own under `vars`. Set `system`
**or** `prompt_template`, not both. Unmatched placeholders are left as-is.

```toml
[[agents]]
name           = "researcher"
provider       = "openai"
model          = "mlx-community/Qwen3-4B-Instruct-2507-8bit"
base_url       = "http://localhost:8080/v1"
prompt_template = "You are {agent}, a research assistant using {model}. Focus on {focus}."
vars           = { focus = "molecular biology" }
```

## Tool calling & reasoning

By default an agent takes the fast path: a single LLM call, no continuation loop.
Set `[agents.tools.reasoning]` to opt an agent into AgenticGoKit's multi-step
tool-calling loop, where the model can call tools and reason over the results
across several turns. The loop — prompt formatting, tool-call parsing, execution,
and continuation — is provided by AgenticGoKit's v1beta pipeline; the orchestrator
only maps this config onto it. No custom loop is reimplemented here.

```toml
[[agents]]
name     = "assistant"
provider = "openai"
model    = "mlx-community/Qwen3-4B-Instruct-2507-8bit"
base_url = "http://localhost:8080/v1"

  [agents.tools.reasoning]
  enabled        = true   # opt into the reasoning loop (default: false → fast path)
  max_iterations = 5      # cap on continuation turns (default: 5 when unset)
  max_concurrent = 1      # max tools executed in parallel per turn
```

Which tools exist is configured once at the top level; `shell` is gated by an
allowlist:

```toml
[tools]
enabled   = ["fetch", "file", "shell"]
allowlist = ["echo", "ls"]
```

MLX tool-calling reliability varies by model family — Qwen and Llama families are
the most consistent. `max_iterations` bounds the loop so a model that keeps
requesting tools cannot spin forever.

For more information, see the [AGENTS.md](AGENTS.md) file.
