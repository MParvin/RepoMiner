package queue_test

import (
	"context"
	"testing"

	"github.com/mparvin/repo-miner/internal/core/domain"
	"github.com/mparvin/repo-miner/internal/core/queue"
)

func TestMemoryQueue(t *testing.T) {
	q := queue.NewMemoryQueue()
	ctx := context.Background()

	job := domain.Job{ID: "test-1", Type: "collect"}
	if err := q.Enqueue(ctx, job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if q.Size() != 1 {
		t.Errorf("expected size 1, got %d", q.Size())
	}

	dequeued, err := q.Dequeue(ctx)
	if err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if dequeued.ID != "test-1" {
		t.Errorf("expected job test-1, got %s", dequeued.ID)
	}
	if q.Size() != 0 {
		t.Errorf("expected size 0, got %d", q.Size())
	}
}

func TestMemoryQueueEmpty(t *testing.T) {
	q := queue.NewMemoryQueue()
	_, err := q.Dequeue(context.Background())
	if err != queue.ErrEmpty {
		t.Errorf("expected ErrEmpty, got %v", err)
	}
}
