package queue

import (
	"context"
	"errors"
	"sync"

	"github.com/mparvin/repo-miner/internal/core/domain"
)

// ErrEmpty is returned when attempting to dequeue from an empty queue.
var ErrEmpty = errors.New("queue is empty")

// Queue abstracts job scheduling for the dataset pipeline.
type Queue interface {
	Enqueue(ctx context.Context, job domain.Job) error
	Dequeue(ctx context.Context) (domain.Job, error)
	Size() int
}

// MemoryQueue is an in-memory FIFO queue implementation.
type MemoryQueue struct {
	mu   sync.Mutex
	jobs []domain.Job
}

// NewMemoryQueue creates a new in-memory queue.
func NewMemoryQueue() *MemoryQueue {
	return &MemoryQueue{
		jobs: make([]domain.Job, 0),
	}
}

// Enqueue adds a job to the back of the queue.
func (q *MemoryQueue) Enqueue(_ context.Context, job domain.Job) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.jobs = append(q.jobs, job)
	return nil
}

// Dequeue removes and returns the job at the front of the queue.
func (q *MemoryQueue) Dequeue(_ context.Context) (domain.Job, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.jobs) == 0 {
		return domain.Job{}, ErrEmpty
	}
	job := q.jobs[0]
	q.jobs = q.jobs[1:]
	return job, nil
}

// Size returns the number of jobs in the queue.
func (q *MemoryQueue) Size() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.jobs)
}
