// Package e2e holds the fixtures and config for the orchestrator's end-to-end
// suite: a sample repo and a repo-agnostic config. The harness (guarded by the
// `e2e` build tag) copies the sample repo to a temp dir, points the orchestrator
// at it, and asks questions it can only answer by reading the code.
package e2e

import "embed"

// ConfigTOML is the committed e2e orchestrator config (see orchestrator.toml).
//
//go:embed orchestrator.toml
var ConfigTOML string

// SampleRepo is the fixture repo tree under sample-repo. The harness
// materializes it to a temp directory to emulate a freshly cloned repo.
//
//go:embed sample-repo
var SampleRepo embed.FS

// SampleRepoDir is the path prefix of the fixture repo inside SampleRepo.
const SampleRepoDir = "sample-repo"
