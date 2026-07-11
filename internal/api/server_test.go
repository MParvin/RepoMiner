package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mparvin/repo-miner/internal/api"
	"github.com/mparvin/repo-miner/internal/core/job"
	"github.com/mparvin/repo-miner/internal/core/queue"
)

func TestHealthEndpoint(t *testing.T) {
	store := job.NewStore()
	srv := api.New(api.Config{Addr: ":8080", JobStore: store, Queue: queue.NewMemoryQueue()})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestCollectJob(t *testing.T) {
	store := job.NewStore()
	q := queue.NewMemoryQueue()
	srv := api.New(api.Config{Addr: ":8080", APIKey: "test-key", JobStore: store, Queue: q})

	body, _ := json.Marshal(map[string]string{"repo": "gin-gonic/gin"})
	req := httptest.NewRequest(http.MethodPost, "/jobs/collect", bytes.NewReader(body))
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["job_id"] == "" {
		t.Error("expected job_id in response")
	}
}
