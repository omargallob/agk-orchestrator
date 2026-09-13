package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunNoArgsShowsUsage(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run(nil, strings.NewReader(""), &out, &errb); code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	for _, c := range commands {
		if !strings.Contains(errb.String(), c) {
			t.Errorf("usage %q missing command %q", errb.String(), c)
		}
	}
}

func TestRunUnknownCommand(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"frobnicate"}, strings.NewReader(""), &out, &errb); code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(errb.String(), "unknown command") {
		t.Errorf("stderr = %q", errb.String())
	}
}
