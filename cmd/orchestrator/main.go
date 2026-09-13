// Command orchestrator is a deployable multi-agent app on AgenticGoKit that uses
// a local MLX model (mlx_lm.server, OpenAI-compatible) as the reasoning brain.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
)

// commands lists the subcommands the CLI supports.
var commands = []string{"run", "serve"}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// run dispatches a subcommand and returns the process exit code. Kept separate
// from main so it is unit-testable.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintf(stderr, "usage: orchestrator <%s> [args]\n", strings.Join(commands, "|"))
		return 2
	}
	switch cmd := args[0]; cmd {
	case "run":
		return cmdRun(context.Background(), args[1:], stdin, stdout, stderr)
	case "serve":
		fmt.Fprintln(stderr, "orchestrator: 'serve' is not yet implemented")
		return 2
	default:
		fmt.Fprintf(stderr, "orchestrator: unknown command %q\n", cmd)
		return 2
	}
}
