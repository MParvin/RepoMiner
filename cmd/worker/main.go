package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/mparvin/repo-miner/internal/config"
	"github.com/mparvin/repo-miner/internal/core/job"
	"github.com/mparvin/repo-miner/internal/core/queue"
	"github.com/mparvin/repo-miner/internal/worker"

	_ "github.com/mparvin/repo-miner/plugins/providers/github"
	_ "github.com/mparvin/repo-miner/plugins/providers/gitlab"
	_ "github.com/mparvin/repo-miner/plugins/providers/localgit"
	_ "github.com/mparvin/repo-miner/plugins/analyzers/golang"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
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
	runner, err := worker.NewRunner(cfg, q, jobStore)
	if err != nil {
		fmt.Fprintf(os.Stderr, "worker: %v\n", err)
		os.Exit(1)
	}

	if err := runner.Run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "worker: %v\n", err)
		os.Exit(1)
	}
}
