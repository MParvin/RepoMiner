package job

import (
	"sync"
	"time"
)

// Status represents job lifecycle state.
type Status string

const (
	StatusPending    Status = "pending"
	StatusRunning    Status = "running"
	StatusCompleted  Status = "completed"
	StatusFailed     Status = "failed"
)

// Record tracks a job's state.
type Record struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Status    Status            `json:"status"`
	Payload   map[string]string `json:"payload"`
	Error     string            `json:"error,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// Store tracks job records.
type Store struct {
	mu    sync.RWMutex
	jobs  map[string]*Record
	stats Stats
}

// Stats holds platform statistics.
type Stats struct {
	Repositories   int `json:"repositories"`
	Analyzed       int `json:"analyzed"`
	DatasetSamples int `json:"dataset_samples"`
	ActiveJobs     int `json:"active_jobs"`
}

// NewStore creates an in-memory job store.
func NewStore() *Store {
	return &Store{jobs: make(map[string]*Record)}
}

// Create registers a new job.
func (s *Store) Create(id, jobType string, payload map[string]string) *Record {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	rec := &Record{
		ID: id, Type: jobType, Status: StatusPending,
		Payload: payload, CreatedAt: now, UpdatedAt: now,
	}
	s.jobs[id] = rec
	s.stats.ActiveJobs++
	return rec
}

// Get returns a job by ID.
func (s *Store) Get(id string) (*Record, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.jobs[id]
	return rec, ok
}

// UpdateStatus updates a job's status.
func (s *Store) UpdateStatus(id string, status Status, errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rec, ok := s.jobs[id]; ok {
		rec.Status = status
		rec.Error = errMsg
		rec.UpdatedAt = time.Now().UTC()
		if status == StatusCompleted || status == StatusFailed {
			s.stats.ActiveJobs--
		}
	}
}

// GetStats returns current platform stats.
func (s *Store) GetStats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stats
}

// SetStats updates platform statistics.
func (s *Store) SetStats(stats Stats) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stats = stats
}

// IncrementRepositories increments the repository count.
func (s *Store) IncrementRepositories() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stats.Repositories++
}

// IncrementAnalyzed increments the analyzed count.
func (s *Store) IncrementAnalyzed() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stats.Analyzed++
}

// AddDatasetSamples adds to the dataset sample count.
func (s *Store) AddDatasetSamples(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stats.DatasetSamples += n
}
