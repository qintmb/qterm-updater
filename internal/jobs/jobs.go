package jobs

import (
	"context"
	"os/exec"
	"sync"
	"time"

	"github.com/qintmb/qterm-updater/internal/scan"
	"github.com/qintmb/qterm-updater/internal/state"
)

// Job is one running upgrade.
type Job struct {
	Pkg     scan.Row
	Cmd     *exec.Cmd
	Started time.Time
	Done    chan struct{}
	Output  []byte
	Err     error
}

// Queue runs upgrades one at a time.
type Queue struct {
	mu     sync.Mutex
	jobs   chan Job
	cancel context.CancelFunc
}

// NewQueue creates a job queue that runs one job at a time.
func NewQueue() *Queue {
	q := &Queue{jobs: make(chan Job, 100)}
	go q.runLoop()
	return q
}

func (q *Queue) runLoop() {
	for j := range q.jobs {
		ctx, cancel := context.WithCancel(context.Background())
		q.mu.Lock()
		q.cancel = cancel
		q.mu.Unlock()
		args := q.cmdFor(j.Pkg)
		j.Cmd = exec.CommandContext(ctx, args[0], args[1:]...)
		j.Cmd.Start()
		j.Output, j.Err = j.Cmd.CombinedOutput() // nolint:fixa
		close(j.Done)
		state.AppendHistory(state.HistoryEntry{
			StartedAt: j.Started,
			EndedAt:   time.Now(),
			Pkg:       j.Pkg.Pkg,
			Manager:   j.Pkg.Manager,
			From:      j.Pkg.Cur,
			To:        j.Pkg.New,
			OK:        j.Err == nil && j.Cmd.ProcessState.Success(),
			Output:    string(j.Output),
		})
	}
}

func (q *Queue) cmdFor(r scan.Row) []string {
	switch r.Manager {
	case "brew":
		return []string{"brew", "upgrade", r.Pkg}
	case "npm":
		return []string{"npm", "install", "-g", r.Pkg + "@latest"}
	case "pnpm":
		return []string{"pnpm", "update", "-g", "--latest", r.Pkg}
	case "uv":
		return []string{"uv", "tool", "upgrade", r.Pkg}
	default:
		return nil
	}
}

// Enqueue queues a package for upgrade and returns the job.
func (q *Queue) Enqueue(r scan.Row) Job {
	j := Job{Pkg: r, Started: time.Now(), Done: make(chan struct{})}
	q.jobs <- j
	return j
}

// Cancel stops the current job, if any.
func (q *Queue) Cancel() {
	q.mu.Lock()
	if q.cancel != nil {
		q.cancel()
	}
	q.mu.Unlock()
}
