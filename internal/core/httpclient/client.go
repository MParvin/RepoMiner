package httpclient

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// RawSaver persists raw API responses for debugging and replay.
type RawSaver interface {
	SaveRaw(ctx context.Context, endpoint string, data []byte) error
}

// Client is an HTTP client with rate limiting, retries, and optional raw response capture.
type Client struct {
	baseURL    string
	token      string
	provider   string
	httpClient *http.Client
	rawSaver   RawSaver
	mu         sync.Mutex
	lastReq    time.Time
	minGap     time.Duration
}

// Config configures the HTTP client.
type Config struct {
	BaseURL  string
	Token    string
	Provider string
	RawSaver RawSaver
	MinGap   time.Duration
}

// New creates a new rate-limited HTTP client.
func New(cfg Config) *Client {
	minGap := cfg.MinGap
	if minGap == 0 {
		minGap = 100 * time.Millisecond
	}
	return &Client{
		baseURL:    cfg.BaseURL,
		token:      cfg.Token,
		provider:   cfg.Provider,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		rawSaver:   cfg.RawSaver,
		minGap:     minGap,
	}
}

// Get performs a GET request with rate limiting and retries.
func (c *Client) Get(ctx context.Context, path string) ([]byte, error) {
	url := c.baseURL + path
	var lastErr error

	for attempt := 0; attempt < 5; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		c.throttle()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json")
		if c.token != "" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}

		if c.rawSaver != nil {
			_ = c.rawSaver.SaveRaw(ctx, path, body)
		}

		switch resp.StatusCode {
		case http.StatusOK:
			return body, nil
		case http.StatusForbidden, http.StatusTooManyRequests:
			if wait := parseRetryAfter(resp); wait > 0 {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(wait):
				}
			}
			lastErr = fmt.Errorf("rate limited (status %d): %s", resp.StatusCode, truncate(body, 200))
			continue
		case http.StatusUnauthorized:
			return nil, fmt.Errorf("unauthorized (status 401): check your token")
		case http.StatusNotFound:
			return nil, fmt.Errorf("not found: %s", path)
		default:
			if resp.StatusCode >= 500 {
				lastErr = fmt.Errorf("server error %d: %s", resp.StatusCode, truncate(body, 200))
				continue
			}
			return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, truncate(body, 200))
		}
	}
	return nil, fmt.Errorf("request failed after retries: %w", lastErr)
}

func (c *Client) throttle() {
	c.mu.Lock()
	defer c.mu.Unlock()
	elapsed := time.Since(c.lastReq)
	if elapsed < c.minGap {
		time.Sleep(c.minGap - elapsed)
	}
	c.lastReq = time.Now()
}

func parseRetryAfter(resp *http.Response) time.Duration {
	if v := resp.Header.Get("Retry-After"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil {
			return time.Duration(secs) * time.Second
		}
	}
	if remaining := resp.Header.Get("X-RateLimit-Remaining"); remaining == "0" {
		if reset := resp.Header.Get("X-RateLimit-Reset"); reset != "" {
			if ts, err := strconv.ParseInt(reset, 10, 64); err == nil {
				wait := time.Until(time.Unix(ts, 0))
				if wait > 0 {
					return wait
				}
			}
		}
	}
	return 2 * time.Second
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}

// StorageRawSaver adapts storage.SaveRawResponse to RawSaver.
type StorageRawSaver struct {
	Store    interface {
		SaveRawResponse(ctx context.Context, provider, endpoint string, data []byte) error
	}
	Provider string
}

// SaveRaw saves a raw response via storage.
func (s *StorageRawSaver) SaveRaw(ctx context.Context, endpoint string, data []byte) error {
	return s.Store.SaveRawResponse(ctx, s.Provider, endpoint, data)
}
