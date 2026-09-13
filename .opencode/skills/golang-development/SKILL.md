---
name: golang-development
description: Enforce samber/cc-skills-golang for all Golang development. Use when writing, reviewing, testing, or refactoring Go code, *.go files, go.mod, Bazel Go targets, or discussing Go conventions, concurrency, errors, and performance. Use ONLY for Go work.
---

# Golang Development (cc-skills-golang)

All Go work in this repo MUST follow
[samber/cc-skills-golang](https://github.com/samber/cc-skills-golang).
That repo is registered as the `cc-skills-golang` reference in
`opencode.json` — load the relevant skill from there (or the matching
global `golang-*` skill) BEFORE writing or reviewing Go code.

## Mandatory routing

Match the task to the skill, then load it via the skill tool
(e.g. `golang-testing`) or `@cc-skills-golang:skills/<name>`:

- Style, clarity, comments → `golang-code-style`
- Identifiers, packages, acronyms → `golang-naming`
- Errors, wrapping, sentinel, `%w` → `golang-error-handling`
- Structs, interfaces, embedding, receivers → `golang-structs-interfaces`
- Table tests, fuzzing, testify, coverage → `golang-testing`
- Benchmarks, pprof, benchstat → `golang-benchmark`
- Allocation, GC, pooling, hot paths → `golang-performance`
- Goroutines, channels, errgroup, pools → `golang-concurrency`
- ctx propagation, timeouts, values → `golang-context`
- Slices, maps, strings, generics → `golang-data-structures`
- Patterns, options, shutdown, retries → `golang-design-patterns`
- SQL, transactions, pooling, migrations → `golang-database`
- Injection, crypto, secrets, SSRF → `golang-security`
- godoc, README, examples → `golang-documentation`
- Modernize, iterators, stdlib upgrades → `golang-modernize`
- Layout, cmd/internal/pkg, workspaces → `golang-project-layout`
- Library choice, stdlib-first → `golang-popular-libraries`

Framework skills (`golang-spf13-cobra`, `golang-grpc`, `golang-samber-*`,
`golang-stretchr-testify`, …) apply when the dependency is already in
`go.mod` — never add a dependency just to match a skill.

## Project overrides (precedence)

Repo rules win on conflict, in this order:

1. `AGENTS.md` — Bazel gate (`bazel test //...`), TDD, `gofmt`, small
   injectable packages, Conventional Commits.
2. This skill — cc-skills-golang conventions for Go idioms.
3. Linter config (`.golangci.yml`) and `go.mod` toolchain.

`gofmt` is non-negotiable; skill advice never overrides a failing
`bazel test //...` or `golangci-lint` run.

## Workflow

1. Classify the task using the routing table above.
2. Load at least the one matching skill; load a second when areas
   overlap (e.g. `golang-error-handling` + `golang-testing`).
3. Apply it, then verify: `gofmt -l .`, `bazel test //...`,
   `bazel run //tools/lint:golangci-lint -- run`.
4. If no cc-skills-golang skill covers the task, say so explicitly —
   do not invent conventions.
