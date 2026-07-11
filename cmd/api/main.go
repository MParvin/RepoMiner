package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/mparvin/repo-miner/internal/api"
	"github.com/mparvin/repo-miner/internal/config"
	"github.com/mparvin/repo-miner/internal/core/job"
	"github.com/mparvin/repo-miner/internal/core/queue"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	addr := flag.String("addr", ":8080", "listen address")
	apiKey := flag.String("api-key", os.Getenv("API_KEY"), "API key for authentication")
	redisAddr := flag.String("redis", os.Getenv("REDIS_ADDR"), "Redis address")
	flag.Parse()

	if *redisAddr == "" {
		*redisAddr = "localhost:6379"
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	var q queue.Queue
	if cfg.Queue.Driver == "redis" {
		rq, err := queue.NewRedisQueue(*redisAddr, "", 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "redis: %v\n", err)
			os.Exit(1)
		}
		defer rq.Close()
		q = rq
	} else {
		q = queue.NewMemoryQueue()
	}

	jobStore := job.NewStore()
	server := api.New(api.Config{
		Addr:     *addr,
		APIKey:   *apiKey,
		JobStore: jobStore,
		Queue:    q,
	})

	if err := server.ListenAndServe(); err != nil {
		fmt.Fprintf(os.Stderr, "server: %v\n", err)
		os.Exit(1)
	}
	_ = context.Background()
}
