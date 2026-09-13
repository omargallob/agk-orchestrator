package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/agenticgokit/agenticgokit/v1beta"
)

type fakeRunner struct {
	workflows []string
	result    *v1beta.WorkflowResult
	err       error
}

func (f *fakeRunner) Workflows() []string { return f.workflows }

func (f *fakeRunner) Run(_ context.Context, _, _ string) (*v1beta.WorkflowResult, error) {
	return f.result, f.err
}

func newTestServer(r Runner) *httptest.Server {
	return httptest.NewServer(New(r, 5*time.Second).Handler())
}

func submit(t *testing.T, base, body string) (Job, int) {
	t.Helper()
	resp, err := http.Post(base+"/jobs", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST /jobs: %v", err)
	}
	defer resp.Body.Close()
	var j Job
	if resp.StatusCode < 300 {
		if err := json.NewDecoder(resp.Body).Decode(&j); err != nil {
			t.Fatalf("decode job: %v", err)
		}
	}
	return j, resp.StatusCode
}

func poll(t *testing.T, base, id string) Job {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(base + "/jobs/" + id)
		if err != nil {
			t.Fatalf("GET job: %v", err)
		}
		var j Job
		json.NewDecoder(resp.Body).Decode(&j)
		resp.Body.Close()
		if j.Status == StatusSucceeded || j.Status == StatusFailed {
			return j
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("job did not reach a terminal status in time")
	return Job{}
}

func TestHealth(t *testing.T) {
	srv := newTestServer(&fakeRunner{})
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("healthz = %d, want 200", resp.StatusCode)
	}
}

func TestSubmitAndSucceed(t *testing.T) {
	r := &fakeRunner{workflows: []string{"main"}, result: &v1beta.WorkflowResult{Success: true, FinalOutput: "done"}}
	srv := newTestServer(r)
	defer srv.Close()

	job, code := submit(t, srv.URL, `{"input":"hi"}`)
	if code != http.StatusAccepted {
		t.Fatalf("submit code = %d, want 202", code)
	}
	if job.ID == "" || job.Workflow != "main" {
		t.Fatalf("unexpected job %+v", job)
	}
	final := poll(t, srv.URL, job.ID)
	if final.Status != StatusSucceeded || final.Output != "done" {
		t.Errorf("final = %+v", final)
	}
	if final.FinishedAt == nil {
		t.Error("FinishedAt not set")
	}
}

func TestSubmitFailure(t *testing.T) {
	r := &fakeRunner{workflows: []string{"main"}, err: errors.New("boom")}
	srv := newTestServer(r)
	defer srv.Close()

	job, code := submit(t, srv.URL, `{"workflow":"main","input":"hi"}`)
	if code != http.StatusAccepted {
		t.Fatalf("submit code = %d", code)
	}
	final := poll(t, srv.URL, job.ID)
	if final.Status != StatusFailed || !strings.Contains(final.Error, "boom") {
		t.Errorf("final = %+v", final)
	}
}

func TestSubmitUnknownWorkflow(t *testing.T) {
	srv := newTestServer(&fakeRunner{workflows: []string{"main"}})
	defer srv.Close()
	_, code := submit(t, srv.URL, `{"workflow":"nope","input":"hi"}`)
	if code != http.StatusBadRequest {
		t.Errorf("code = %d, want 400", code)
	}
}

func TestSubmitMissingInput(t *testing.T) {
	srv := newTestServer(&fakeRunner{workflows: []string{"main"}})
	defer srv.Close()
	_, code := submit(t, srv.URL, `{"workflow":"main"}`)
	if code != http.StatusBadRequest {
		t.Errorf("code = %d, want 400", code)
	}
}

func TestGetUnknownJob(t *testing.T) {
	srv := newTestServer(&fakeRunner{})
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/jobs/deadbeef")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("code = %d, want 404", resp.StatusCode)
	}
}
