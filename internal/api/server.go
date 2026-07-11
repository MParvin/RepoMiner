package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mparvin/repo-miner/internal/core/domain"
	"github.com/mparvin/repo-miner/internal/core/job"
	"github.com/mparvin/repo-miner/internal/core/queue"
)

// Server is the REST API server.
type Server struct {
	addr     string
	apiKey   string
	jobStore *job.Store
	queue    queue.Queue
	mux      *http.ServeMux
}

// Config configures the API server.
type Config struct {
	Addr     string
	APIKey   string
	JobStore *job.Store
	Queue    queue.Queue
}

// New creates a new API server.
func New(cfg Config) *Server {
	s := &Server{
		addr:     cfg.Addr,
		apiKey:   cfg.APIKey,
		jobStore: cfg.JobStore,
		queue:    cfg.Queue,
		mux:      http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /stats", s.auth(s.handleStats))
	s.mux.HandleFunc("POST /jobs/collect", s.auth(s.handleCollect))
	s.mux.HandleFunc("GET /jobs/{id}/status", s.auth(s.handleJobStatus))
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe() error {
	fmt.Printf("API server listening on %s\n", s.addr)
	return http.ListenAndServe(s.addr, s.mux)
}

// ServeHTTP implements http.Handler for testing.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.apiKey != "" {
			key := r.Header.Get("X-API-Key")
			if key == "" {
				key = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			}
			if key != s.apiKey {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}
		next(w, r)
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) handleStats(w http.ResponseWriter, _ *http.Request) {
	stats := s.jobStore.GetStats()
	if s.queue != nil {
		stats.ActiveJobs = s.queue.Size()
	}
	writeJSON(w, stats)
}

type collectRequest struct {
	Repo     string `json:"repo"`
	Provider string `json:"provider"`
}

func (s *Server) handleCollect(w http.ResponseWriter, r *http.Request) {
	var req collectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Repo == "" {
		http.Error(w, "repo is required", http.StatusBadRequest)
		return
	}

	jobID := fmt.Sprintf("collect-%s-%d", strings.ReplaceAll(req.Repo, "/", "-"), time.Now().Unix())
	payload := map[string]string{"repo": req.Repo}
	if req.Provider != "" {
		payload["provider"] = req.Provider
	}

	s.jobStore.Create(jobID, "collect", payload)

	domainJob := domain.Job{
		ID:      jobID,
		Type:    "collect",
		Payload: payload,
	}
	if err := s.queue.Enqueue(r.Context(), domainJob); err != nil {
		s.jobStore.UpdateStatus(jobID, job.StatusFailed, err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"job_id": jobID, "status": "queued"})
}

func (s *Server) handleJobStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	rec, ok := s.jobStore.Get(id)
	if !ok {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}
	writeJSON(w, rec)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// UpdateJobStatus is called by workers to update job state.
func UpdateJobStatus(store *job.Store, id string, status job.Status, errMsg string) {
	store.UpdateStatus(id, status, errMsg)
}

// ProcessCollectJob is the collect job handler used by workers.
func ProcessCollectJob(ctx context.Context, _ *job.Store, _ domain.Job) error {
	_ = ctx
	return nil // wired in worker package
}
