# Skills

Portable [SKILL.md](https://agentskills.io) packages distilled from this repo's
workflows. They are repo-agnostic and install into Claude Code, Cursor, Codex
CLI and other agents that support the SKILL.md format.

| Skill | Use it for |
|---|---|
| [bazel-go-hygiene](bazel-go-hygiene/SKILL.md) | Keeping a Bazel + Go (rules_go, gazelle, bzlmod) repo green: gazelle, `bazel mod tidy`, nogo, buildifier, pre-PR checks |
| [local-mlx-agent-e2e](local-mlx-agent-e2e/SKILL.md) | Running `mlx_lm.server` as an agent's brain and testing tool calling with a scripted fake + env-gated live harness |

Install for Claude Code (per user):

```bash
cp -R skills/bazel-go-hygiene skills/local-mlx-agent-e2e ~/.claude/skills/
```
