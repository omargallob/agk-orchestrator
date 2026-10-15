---
name: bazel-go-hygiene
description: Keep a Bazel + Go (rules_go, gazelle, bzlmod) repository green. Use when adding or removing Go files or packages, changing go.mod, adding a Go dependency, seeing gazelle/BUILD drift, nogo or go vet failures inside a Bazel build, buildifier errors, "no such package @com_github_..." errors, use_repo warnings, or before opening a PR in a Bazel Go repo.
---

# Bazel + Go hygiene

In a Bazel Go repo, `go build` / `go test` passing proves very little: Bazel
builds from BUILD files, not from `go.mod` + directory layout. These steps keep
the two worlds in sync and keep CI green.

## The gate

`bazel test //...` is the source of truth. If the repo wires
`go_sdk.nogo(nogo = "//:nogo")` in `MODULE.bazel`, static analysis (go vet
analyzers) runs **inside every build** — a vet finding is a build failure, not
a lint warning. Don't "fix" it by running `go vet` separately; fix the code.

`go build ./...` / `go test ./...` are fine for a fast inner loop, but never
treat them as the gate.

## After each kind of change

| You changed… | Run |
|---|---|
| Added/removed/renamed a `.go` file, or changed imports | `bazel run //:gazelle` |
| `go.mod` (added, upgraded, or removed a dependency) | `go mod tidy && bazel mod tidy && bazel run //:gazelle` |
| Any `BUILD.bazel`, `.bzl`, or `MODULE.bazel` | `bazel run //:buildifier` (or the repo's buildifier target) |
| Anything | `bazel test //...` |

Why each step:

- **`go mod tidy`** — makes `go.mod`/`go.sum` match the imports.
- **`bazel mod tidy`** — rewrites the `use_repo(go_deps, ...)` list in
  `MODULE.bazel` to match `go.mod`. Skipping it gives
  `no such package '@com_github_foo_bar//...'` or "imported but not used"
  `use_repo` warnings. Never hand-edit that `use_repo` list.
- **gazelle** — regenerates `go_library` / `go_test` targets, `deps`,
  `embedsrcs`, etc. If you added a `//go:embed` file, gazelle picks it up;
  hand-written `embedsrcs` drift.

## Gotchas

- **Dependency ships its own BUILD/MODULE files** and they're broken or
  incompatible: tell gazelle to regenerate them.

  ```starlark
  go_deps.gazelle_override(
      path = "github.com/example/dep",
      build_file_generation = "clean",
  )
  ```

- **`select()` on OS/CPU** (`@platforms//os:linux`, `@platforms//cpu:arm64`)
  requires `bazel_dep(name = "platforms", ...)` as a **direct** dependency in
  `MODULE.bazel`. A transitive one is not visible to your module.

- **Build tags** (e.g. a `//go:build e2e` test): gazelle won't know which tags
  you want. Set `gotags = ["e2e"]` on the `go_test`, and add
  `tags = ["manual"]` so `bazel test //...` doesn't run it by default. Run it
  explicitly with `bazel test //path:target`.

- **Hermetic Go SDK**: if `MODULE.bazel` has `go_sdk.download(version = ...)`,
  Bazel ignores your host Go. Tools that typecheck with the host toolchain
  (golangci-lint, gopls) need a host Go matching `go.mod`.

- **Stray artifacts**: keep `/bazel-*` symlinks and any `go build` output
  binary gitignored.

## Pre-PR checklist

Mirror what CI runs; run all of these locally before pushing:

```bash
bazel test //...                              # build + nogo + tests
bazel run //:gazelle -- -mode=diff            # BUILD files up to date (no diff)
bazel run //:buildifier.check                 # BUILD/.bzl formatted (if defined)
```

Plus the repo's Go linter if it has one (e.g. a hermetic
`bazel run //tools/lint:golangci-lint -- run`).

If `gazelle -mode=diff` prints a diff, run `bazel run //:gazelle`, commit the
result, and re-run.

## Minimal root BUILD.bazel for reference

```starlark
load("@gazelle//:def.bzl", "gazelle")
load("@rules_go//go:def.bzl", "nogo")

# gazelle:prefix github.com/OWNER/REPO
gazelle(name = "gazelle")

nogo(
    name = "nogo",
    vet = True,
    visibility = ["//visibility:public"],
)
```

And in `MODULE.bazel`:

```starlark
go_sdk = use_extension("@rules_go//go:extensions.bzl", "go_sdk")
go_sdk.download(version = "1.x.y")
go_sdk.nogo(nogo = "//:nogo")

go_deps = use_extension("@gazelle//:extensions.bzl", "go_deps")
go_deps.from_file(go_mod = "//:go.mod")
use_repo(go_deps)  # maintained by `bazel mod tidy`
```
