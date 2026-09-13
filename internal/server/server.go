// Package server exposes the orchestrator over HTTP as an async job API:
// POST /jobs submits a workflow run, GET /jobs/{id} polls its status/result,
// and GET /healthz is a liveness check.
package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/agenticgokit/agenticgokit/v1beta"
)

// Job status values.
const (
	StatusQueued    = "queued"
	StatusRunning   = "running"
	StatusSucceeded = "succeeded"
	StatusFailed    = "failed"
)

// Runner executes named workflows. *orchestrator.Orchestrator satisfies this;
// tests use a fake so the HTTP layer needs no network.
type Runner interface {
	Run(ctx context.Context, workflow, input string) (*v1beta.WorkflowResult, error)
	Workflows() []string
}

// Job is a single async workflow execution.
type Job struct {
	ID         string     `json:"id"`
	Workflow   string     `json:"workflow"`
	Status     string     `json:"status"`
	Output     string     `json:"output,omitempty"`
	Error      string     `json:"error,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
}

// Server holds the runner and the in-memory job store.
type Server struct {
	runner  Runner
	timeout time.Duration

	mu   sync.Mutex
	jobs map[string]*Job
}

// New returns a Server. jobTimeout bounds each workflow run (0 = no timeout).
func New(runner Runner, jobTimeout time.Duration) *Server {
	return &Server{
		runner:  runner,
		timeout: jobTimeout,
		jobs:    make(map[string]*Job),
	}
}

// Handler returns the HTTP routes.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("POST /jobs", s.handleSubmit)
	mux.HandleFunc("GET /jobs/{id}", s.handleGet)
	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "ok")
}

type submitRequest struct {
	Workflow string `json:"workflow"`
	Input    string `json:"input"`
}

func (s *Server) handleSubmit(w http.ResponseWriter, r *http.Request) {
	var req submitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Input == "" {
		writeError(w, http.StatusBadRequest, "input is required")
		return
	}
	wf, err := s.resolveWorkflow(req.Workflow)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	job := &Job{ID: newID(), Workflow: wf, Status: StatusQueued, CreatedAt: time.Now()}
	s.mu.Lock()
	s.jobs[job.ID] = job
	s.mu.Unlock()

	go s.execute(job, req.Input)

	w.Header().Set("Location", "/jobs/"+job.ID)
	writeJSON(w, http.StatusAccepted, job)
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s.mu.Lock()
	job, ok := s.jobs[id]
	var snapshot Job
	if ok {
		snapshot = *job
	}
	s.mu.Unlock()
	if !ok {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}
	writeJSON(w, http.StatusOK, &snapshot)
}

// execute runs the workflow and updates the job's status/result.
func (s *Server) execute(job *Job, input string) {
	s.setStatus(job, StatusRunning, "", "")

	ctx := context.Background()
	if s.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.timeout)
		defer cancel()
	}

	result, err := s.runner.Run(ctx, job.Workflow, input)
	switch {
	case err != nil:
		s.setStatus(job, StatusFailed, "", err.Error())
	case result != nil && !result.Success:
		s.setStatus(job, StatusFailed, result.FinalOutput, result.Error)
	default:
		s.setStatus(job, StatusSucceeded, result.FinalOutput, "")
	}
}

func (s *Server) setStatus(job *Job, status, output, errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job.Status = status
	if output != "" {
		job.Output = output
	}
	if errMsg != "" {
		job.Error = errMsg
	}
	if status == StatusSucceeded || status == StatusFailed {
		now := time.Now()
		job.FinishedAt = &now
	}
}

// resolveWorkflow picks the requested workflow, or the single one if unspecified.
func (s *Server) resolveWorkflow(requested string) (string, error) {
	available := s.runner.Workflows()
	if requested != "" {
		for _, w := range available {
			if w == requested {
				return requested, nil
			}
		}
		return "", fmt.Errorf("unknown workflow %q", requested)
	}
	switch len(available) {
	case 0:
		return "", fmt.Errorf("no workflows configured")
	case 1:
		return available[0], nil
	default:
		return "", fmt.Errorf("multiple workflows configured; specify one")
	}
}

func newID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
