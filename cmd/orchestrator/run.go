package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/omargallob/agk-orchestrator/internal/config"
	"github.com/omargallob/agk-orchestrator/internal/orchestrator"
)

// cmdRun implements `orchestrator run`: load config, build the orchestrator,
// execute a workflow with the given input, and print the result.
func cmdRun(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(stderr)
	configPath := fs.String("config", "", "path to the orchestrator TOML config (required)")
	workflow := fs.String("workflow", "", "workflow to run (default: the only one if a single workflow is defined)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	o, wf, input, err := resolveRun(*configPath, *workflow, fs.Args(), stdin)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	result, err := o.Run(ctx, wf, input)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	if !result.Success {
		fmt.Fprintf(stderr, "workflow %q failed: %s\n", wf, result.Error)
		return 1
	}
	fmt.Fprintln(stdout, result.FinalOutput)
	return 0
}

// resolveRun loads config, builds the orchestrator, and resolves the workflow
// and input. Separated from network execution so it is unit-testable.
func resolveRun(configPath, workflow string, promptArgs []string, stdin io.Reader) (*orchestrator.Orchestrator, string, string, error) {
	if configPath == "" {
		return nil, "", "", fmt.Errorf("--config is required")
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, "", "", err
	}
	o, err := orchestrator.New(cfg)
	if err != nil {
		return nil, "", "", err
	}

	wf, err := resolveWorkflow(o.Workflows(), workflow)
	if err != nil {
		return nil, "", "", err
	}

	input := strings.TrimSpace(strings.Join(promptArgs, " "))
	if input == "" {
		b, err := io.ReadAll(stdin)
		if err != nil {
			return nil, "", "", err
		}
		input = strings.TrimSpace(string(b))
	}
	if input == "" {
		return nil, "", "", fmt.Errorf("no input: pass it as an argument or on stdin")
	}
	return o, wf, input, nil
}

// resolveWorkflow picks the workflow to run: the requested one if given, else the
// single defined workflow, else an error.
func resolveWorkflow(available []string, requested string) (string, error) {
	if requested != "" {
		for _, w := range available {
			if w == requested {
				return requested, nil
			}
		}
		return "", fmt.Errorf("unknown workflow %q (available: %s)", requested, strings.Join(available, ", "))
	}
	switch len(available) {
	case 0:
		return "", fmt.Errorf("config defines no workflows")
	case 1:
		return available[0], nil
	default:
		return "", fmt.Errorf("multiple workflows defined; choose one with --workflow (available: %s)", strings.Join(available, ", "))
	}
}
