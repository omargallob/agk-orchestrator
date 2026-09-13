// Command orchestrator is a deployable multi-agent app on AgenticGoKit that uses
// a local MLX model (mlx_lm.server, OpenAI-compatible) as the reasoning brain.
//
// This is the scaffold entrypoint; the `run` (CLI) and `serve` (HTTP) commands
// land in follow-up issues (#7, #8).
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	_ "github.com/agenticgokit/agenticgokit/v1beta"
)

// commands lists the subcommands the CLI will support.
var commands = []string{"run", "serve"}

func main() {
	os.Exit(run(os.Args[1:], os.Stderr))
}

// run dispatches a subcommand and returns the process exit code. Kept separate
// from main so it is unit-testable.
func run(args []string, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintf(stderr, "usage: orchestrator <%s> [args]\n", strings.Join(commands, "|"))
		return 2
	}
	fmt.Fprintf(stderr, "orchestrator: %q not yet implemented\n", args[0])
	return 2
}
