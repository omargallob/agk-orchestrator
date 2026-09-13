package prompt

import "testing"

func TestResolve(t *testing.T) {
	got := Resolve("You are {agent} using {model}.", map[string]string{
		"agent": "researcher",
		"model": "qwen",
	})
	if got != "You are researcher using qwen." {
		t.Errorf("Resolve = %q", got)
	}
}

func TestResolveLeavesUnknownPlaceholders(t *testing.T) {
	got := Resolve("Hi {agent}, focus on {topic}.", map[string]string{"agent": "a"})
	if got != "Hi a, focus on {topic}." {
		t.Errorf("Resolve = %q", got)
	}
}

func TestResolveNoVarsOrEmpty(t *testing.T) {
	if got := Resolve("plain", nil); got != "plain" {
		t.Errorf("Resolve(no vars) = %q", got)
	}
	if got := Resolve("", map[string]string{"a": "b"}); got != "" {
		t.Errorf("Resolve(empty) = %q", got)
	}
}
