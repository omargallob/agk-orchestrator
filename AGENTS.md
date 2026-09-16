# AGENTS.md

## Project
AI agent orchestrator using AgenticGoKit v1beta. Configuration-driven with TOML files. Agents are built from configuration and executed as workflows.

## Golden rules
- **Configuration-first**: Agents and workflows are defined in TOML config files
- **Validation**: Configuration is validated for required fields and referential integrity
- **Template system**: System prompts support {agent}, {model}, {provider}, {base_url} variables
- **No duplicates**: Agent and workflow names must be unique
- **Provider constraints**: Only openai or vllm providers supported
- **Build/test via Bazel**: `bazel test //...` is the gate (nogo/go-vet runs inside the build). `go build`/`go test` are a convenience only.
- **TDD**: write the failing test first, then the minimum code to pass.
- **Keep it green**: `bazel test //...`, golangci-lint, buildifier, and gazelle-diff must all pass before a PR.

## Agent structure
Each agent is defined by:
- Name: Unique identifier
- Provider: openai or vllm
- Model: Specific model name
- Base URL: mlx_lm.server endpoint
- System prompt: Static string or template with variables
- Vars: Custom key-value pairs for agent context

## Usage
Agents are referenced in workflows by name. Workflows can be sequential or parallel:
- Sequential: Steps executed in order
- Parallel: Steps executed simultaneously

## Configuration examples
```toml
[[agents]]
name = "research_agent"
provider = "openai"
model = "gpt-4"
base_url = "http://localhost:8080/v1"
system = "You are a research assistant..."
```

## Validation rules
- All agents require provider, model, and base_url
- System prompt and prompt_template are mutually exclusive
- Agent names must be unique
- Workflow steps must reference existing agents
- Workflow types must be sequential or parallel
- All agents must have a valid provider (openai or vllm)
- All agents require a model and base_url
- System prompt can be static or templated with variables {agent}, {model}, {provider}, {base_url}
- Agents with both System and PromptTemplate are invalid
- Duplicate agent names are not allowed

## Commands
- Test:            `bazel test //...`
- Lint (Go):       `bazel run //tools/lint:golangci-lint -- run`
- Format BUILD:    `bazel run //:buildifier`
- Regenerate BUILD:`bazel run //:gazelle`   (after adding/removing Go files)
- After go.mod change: `go mod tidy && bazel mod tidy && bazel run //:gazelle`

## Code style
- **Golang skills (mandatory)**: load the relevant `samber/cc-skills-golang` skill before writing or reviewing Go code — see `.opencode/skills/golang-development/SKILL.md` for the task→skill routing (also registered as the `cc-skills-golang` reference in `opencode.json`).
- Match the surrounding code (naming, comment density, idioms); `gofmt` always.
- Small, focused packages. Make logic testable by injecting deps (e.g. handlers take `io.Writer` + an interface), so tests need no network.
- Don't commit binaries; keep `/bazel-*` and build artifacts gitignored.

## Git & PRs
- **Conventional Commits** (`feat:`, `fix:`, `chore:`, `docs:`, `build:`, `ci:`, `test:`, `refactor:`) — Release Please derives versions/CHANGELOG from history.
- One branch + PR per issue; **squash-merge**; close issues with `Closes #N` (repeat the keyword per issue — GitHub only auto-closes the first in a list).

## Bazel gotchas
- External Go deps come via gazelle `go_deps` (from `go.mod`); run `bazel mod tidy` to fix `use_repo`.
- If a dependency ships its own BUILD/MODULE files, regenerate them:
  `go_deps.gazelle_override(path=..., build_file_generation="clean")`.
- `@platforms` must be a direct `bazel_dep` to use os/cpu `select()`s.

## Verify changes end-to-end
Run the app, not just tests, when a change should be user-visible (e.g. `bazel run //cmd/... -- ...`).

<!-- spec-kitty:orientation -->
**Spec Kitty v3.2.7** — project: unknown (healthy)

Two usage patterns:
- **Full mission** (spec → plan → tasks → implement → review → merge):
  trigger: "spec out", "create a mission", "write a spec", "plan this"
  → run `/spec-kitty.specify`
- **Lightweight dispatch** (ad-hoc fix, question, or advice — no mission created):
  trigger: "hey spec kitty", "use spec kitty to", "spec kitty <anything>"
  → **ALWAYS run `spec-kitty dispatch "<request verbatim>"` — do NOT answer directly.**
  If you know the right profile, pass it to skip routing:
  `spec-kitty dispatch "<request verbatim>" --profile <profile-id>`
  Reason: `spec-kitty dispatch` loads governance context, routes the request,
  and opens the Op. Skipping it produces ungoverned, untracked responses.
  After finishing the work, close the Op with the command printed in the capsule
  (`spec-kitty profile-invocation complete --invocation-id <id> --outcome <done|failed|abandoned>`).
<!-- /spec-kitty:orientation -->
