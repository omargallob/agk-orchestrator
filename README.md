# agk-orchestrator

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
