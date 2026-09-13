package e2e

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/omargallob/agk-orchestrator/internal/config"
)

func TestEmbeddedConfigIsValid(t *testing.T) {
	cfg, err := config.Parse([]byte(ConfigTOML))
	if err != nil {
		t.Fatalf("e2e config does not parse/validate: %v", err)
	}

	// The suite depends on the repo-exploration tools being enabled.
	enabled := strings.Join(cfg.Tools.Enabled, ",")
	for _, want := range []string{"list", "grep", "file"} {
		if !strings.Contains(enabled, want) {
			t.Errorf("e2e config must enable %q tool; enabled=%q", want, enabled)
		}
	}
	// And on the agent running the reasoning loop so it can call those tools.
	if len(cfg.Agents) == 0 || !cfg.Agents[0].Tools.Reasoning.Enabled {
		t.Error("e2e config must enable reasoning on the agent")
	}
}

func TestSampleRepoContainsKnownAnswer(t *testing.T) {
	b, err := fs.ReadFile(SampleRepo, SampleRepoDir+"/pricing.py")
	if err != nil {
		t.Fatalf("sample repo missing pricing.py: %v", err)
	}
	if !strings.Contains(string(b), "BASE_PRICE_CENTS = 1999") {
		t.Error("fixture drifted: pricing.py must define BASE_PRICE_CENTS = 1999 (the asserted answer)")
	}
}
