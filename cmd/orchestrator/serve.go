package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/omargallob/agk-orchestrator/internal/config"
	"github.com/omargallob/agk-orchestrator/internal/orchestrator"
	"github.com/omargallob/agk-orchestrator/internal/server"
)

// cmdServe implements `orchestrator serve`: a long-running HTTP async job API.
func cmdServe(args []string, stderr io.Writer) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(stderr)
	configPath := fs.String("config", "", "path to the orchestrator TOML config (required)")
	addr := fs.String("addr", ":8090", "HTTP listen address")
	jobTimeout := fs.Duration("job-timeout", 5*time.Minute, "per-job execution timeout")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *configPath == "" {
		fmt.Fprintln(stderr, "error: --config is required")
		return 1
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	o, err := orchestrator.New(cfg)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	httpSrv := &http.Server{
		Addr:    *addr,
		Handler: server.New(o, *jobTimeout).Handler(),
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		fmt.Fprintf(stderr, "orchestrator serving on %s (workflows: %v)\n", *addr, o.Workflows())
		errCh <- httpSrv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutdownCtx)
		return 0
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
		return 0
	}
}
