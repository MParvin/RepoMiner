package pipeline

import (
	"context"
	"fmt"

	"github.com/mparvin/repo-miner/internal/analyzer"
	"github.com/mparvin/repo-miner/internal/core/domain"
	"github.com/mparvin/repo-miner/internal/core/provider"
	"github.com/mparvin/repo-miner/internal/core/queue"
	"github.com/mparvin/repo-miner/internal/storage"
)

// Stage represents a step in the dataset pipeline.
type Stage string

const (
	StageCollect  Stage = "collect"
	StageAnalyze  Stage = "analyze"
	StageGenerate Stage = "generate"
)

// Pipeline orchestrates the collect -> analyze -> generate workflow.
type Pipeline struct {
	Provider provider.RepositoryProvider
	Analyzer analyzer.LanguageAnalyzer
	Storage  storage.Storage
	Queue    queue.Queue
}

// New creates a new dataset pipeline.
func New(prov provider.RepositoryProvider, ana analyzer.LanguageAnalyzer, store storage.Storage, q queue.Queue) *Pipeline {
	return &Pipeline{
		Provider: prov,
		Analyzer: ana,
		Storage:  store,
		Queue:    q,
	}
}

// Run executes the full pipeline for a repository reference.
func (p *Pipeline) Run(ctx context.Context, ref domain.RepositoryRef) error {
	if p.Provider == nil {
		return fmt.Errorf("pipeline: no provider configured")
	}
	if p.Storage == nil {
		return fmt.Errorf("pipeline: no storage configured")
	}

	job := domain.Job{
		ID:   fmt.Sprintf("collect-%s/%s", ref.Owner, ref.Name),
		Type: string(StageCollect),
		Payload: map[string]string{
			"provider": ref.Provider,
			"owner":    ref.Owner,
			"name":     ref.Name,
		},
	}
	if p.Queue != nil {
		if err := p.Queue.Enqueue(ctx, job); err != nil {
			return fmt.Errorf("enqueue collect job: %w", err)
		}
	}

	repo, err := p.Provider.GetRepository(ctx, ref)
	if err != nil {
		return fmt.Errorf("collect repository: %w", err)
	}
	if err := p.Storage.SaveRepository(ctx, repo); err != nil {
		return fmt.Errorf("save repository: %w", err)
	}

	if p.Analyzer != nil {
		analyzeJob := domain.Job{
			ID:   fmt.Sprintf("analyze-%s/%s", ref.Owner, ref.Name),
			Type: string(StageAnalyze),
			Payload: map[string]string{
				"owner": ref.Owner,
				"name":  ref.Name,
			},
		}
		if p.Queue != nil {
			_ = p.Queue.Enqueue(ctx, analyzeJob)
		}
	}

	return nil
}
