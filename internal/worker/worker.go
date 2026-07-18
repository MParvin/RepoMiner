package worker

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mparvin/repo-miner/internal/collector"
	"github.com/mparvin/repo-miner/internal/config"
	"github.com/mparvin/repo-miner/internal/core/domain"
	"github.com/mparvin/repo-miner/internal/core/job"
	"github.com/mparvin/repo-miner/internal/core/queue"
	"github.com/mparvin/repo-miner/internal/dataset"
	"github.com/mparvin/repo-miner/internal/generator"
	"github.com/mparvin/repo-miner/internal/storage/sqlite"
)

// Runner processes jobs from the queue.
type Runner struct {
	Queue    queue.Queue
	JobStore *job.Store
	Config   *config.Config
	Storage  *sqlite.Storage
}

// NewRunner creates a worker runner.
func NewRunner(cfg *config.Config, q queue.Queue, js *job.Store) (*Runner, error) {
	store, err := sqlite.Open(cfg.Storage.Path)
	if err != nil {
		return nil, err
	}
	if err := store.Migrate(context.Background()); err != nil {
		store.Close()
		return nil, err
	}
	return &Runner{Queue: q, JobStore: js, Config: cfg, Storage: store}, nil
}

// Run starts the worker loop.
func (r *Runner) Run(ctx context.Context) error {
	defer r.Storage.Close()
	fmt.Println("Worker started, waiting for jobs...")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-sigCh:
			fmt.Println("Worker shutting down...")
			return nil
		default:
		}

		domainJob, err := r.Queue.Dequeue(ctx)
		if err != nil {
			if err == queue.ErrEmpty {
				time.Sleep(1 * time.Second)
				continue
			}
			return err
		}

		r.JobStore.UpdateStatus(domainJob.ID, job.StatusRunning, "")
		if err := r.processJob(ctx, domainJob); err != nil {
			r.JobStore.UpdateStatus(domainJob.ID, job.StatusFailed, err.Error())
			fmt.Printf("Job %s failed: %v\n", domainJob.ID, err)
		} else {
			r.JobStore.UpdateStatus(domainJob.ID, job.StatusCompleted, "")
			fmt.Printf("Job %s completed\n", domainJob.ID)
		}
	}
}

func (r *Runner) processJob(ctx context.Context, j domain.Job) error {
	switch j.Type {
	case "collect":
		return r.handleCollect(ctx, j)
	case "analyze":
		return r.handleAnalyze(ctx, j)
	case "generate":
		return r.handleGenerate(ctx, j)
	default:
		return fmt.Errorf("unknown job type: %s", j.Type)
	}
}

func (r *Runner) handleCollect(ctx context.Context, j domain.Job) error {
	repo := j.Payload["repo"]
	providerName := r.Config.Source.Type
	if p, ok := j.Payload["provider"]; ok && p != "" {
		providerName = p
	}

	ref, err := collector.ParseRepoRef(providerName, repo)
	if err != nil {
		return err
	}

	c, err := collector.New(r.Config, r.Storage, ref)
	if err != nil {
		return err
	}
	if err := c.Collect(ctx, ref); err != nil {
		return err
	}
	r.JobStore.IncrementRepositories()
	return nil
}

func (r *Runner) handleAnalyze(ctx context.Context, j domain.Job) error {
	_ = ctx
	_ = j
	r.JobStore.IncrementAnalyzed()
	return nil
}

func (r *Runner) handleGenerate(ctx context.Context, j domain.Job) error {
	repo := j.Payload["repo"]
	ref, err := collector.ParseRepoRef(r.Config.Source.Type, repo)
	if err != nil {
		return err
	}
	gen, err := generator.New(r.Config, r.Storage)
	if err != nil {
		return err
	}
	output := j.Payload["output"]
	name := dataset.ResolveName(j.Payload["name"], j.Payload["keywords"])
	if output == "" {
		paths := dataset.NewPaths(r.Config.Workspace.DatasetsDir, name)
		output = paths.JSONL
	}
	count, err := gen.Generate(ctx, ref, output, "jsonl")
	if err != nil {
		return err
	}
	r.JobStore.AddDatasetSamples(count)
	return nil
}
