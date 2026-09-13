package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunNoArgsShowsUsage(t *testing.T) {
	var buf bytes.Buffer
	if code := run(nil, &buf); code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	out := buf.String()
	for _, c := range commands {
		if !strings.Contains(out, c) {
			t.Errorf("usage %q missing command %q", out, c)
		}
	}
}
