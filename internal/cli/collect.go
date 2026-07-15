package cli

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/mparvin/repo-miner/internal/collector"
	"github.com/mparvin/repo-miner/internal/config"
	"github.com/mparvin/repo-miner/internal/core/domain"
	"github.com/mparvin/repo-miner/internal/dataset"
	"github.com/mparvin/repo-miner/internal/core/provider"
	"github.com/mparvin/repo-miner/internal/storage/sqlite"
)

var collectFlags struct {
	repo          string
	query         string
	keywords      string
	language      string
	createdAfter  string
	createdBefore string
	minStars      int
	maxStars      int
	topic         string
	user          string
	org           string
	forks         string
	archived      string
	sort          string
	order         string
	limit         int
	dryRun        bool
	name          string
}

var collectCmd = &cobra.Command{
	Use:   "collect",
	Short: "Collect repository data from the configured provider",
	Long: `Collect data from a single repository or search and collect multiple repos.

Single repo:
  dataset-builder collect --repo gin-gonic/gin

GitHub-style search (like github.com/search):
  dataset-builder collect --keywords gin --language Go --created-after 2026-01-01
  dataset-builder collect --query "gin created:>2026-01-01 language:Go" --limit 10
  dataset-builder collect --language Go --min-stars 100 --sort stars --limit 5`,
	RunE: runCollect,
}

func init() {
	f := collectCmd.Flags()
	f.StringVar(&collectFlags.repo, "repo", "", "repository to collect (owner/name)")
	f.StringVar(&collectFlags.query, "query", "", "raw search query (GitHub search syntax)")
	f.StringVar(&collectFlags.keywords, "keywords", "", "free-text search terms")
	f.StringVarP(&collectFlags.language, "language", "l", "", "filter by language (e.g. Go, Python)")
	f.StringVar(&collectFlags.createdAfter, "created-after", "", "created after date (YYYY-MM-DD)")
	f.StringVar(&collectFlags.createdBefore, "created-before", "", "created before date (YYYY-MM-DD)")
	f.IntVar(&collectFlags.minStars, "min-stars", 0, "minimum star count")
	f.IntVar(&collectFlags.maxStars, "max-stars", 0, "maximum star count")
	f.StringVar(&collectFlags.topic, "topic", "", "filter by topic")
	f.StringVar(&collectFlags.user, "user", "", "filter by user")
	f.StringVar(&collectFlags.org, "org", "", "filter by organization")
	f.StringVar(&collectFlags.forks, "forks", "", "include forks (true/false)")
	f.StringVar(&collectFlags.archived, "archived", "", "include archived (true/false)")
	f.StringVar(&collectFlags.sort, "sort", "stars", "sort by: stars, updated, forks")
	f.StringVar(&collectFlags.order, "order", "desc", "sort order: desc, asc")
	f.IntVar(&collectFlags.limit, "limit", 30, "max repositories to collect from search")
	f.BoolVar(&collectFlags.dryRun, "dry-run", false, "list search results without collecting")
	f.StringVar(&collectFlags.name, "name", "", "dataset name (used for output files and directories in generate/refine)")
	rootCmd.AddCommand(collectCmd)
}

func runCollect(_ *cobra.Command, _ []string) error {
	cfg := loadConfigOrExit()
	ctx := context.Background()

	store, err := sqlite.Open(cfg.Storage.Path)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer store.Close()

	if err := store.Migrate(ctx); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}

	if collectFlags.repo != "" {
		return collectOne(ctx, cfg, store, collectFlags.repo)
	}

	searchOpts := buildSearchOptions()
	if !collector.HasSearchCriteria(searchOpts) {
		return fmt.Errorf("specify --repo or search flags (--query, --keywords, --language, --created-after, etc.)")
	}

	return collectSearch(ctx, cfg, store, searchOpts)
}

func collectOne(ctx context.Context, cfg *config.Config, store *sqlite.Storage, repo string) error {
	paths, err := dataset.EnsureDir(cfg.Workspace.DatasetsDir, collectFlags.name, collectFlags.keywords)
	if err != nil {
		return fmt.Errorf("create dataset dir: %w", err)
	}
	fmt.Printf("Dataset: %s (dir: %s)\n", paths.Name, paths.Dir)

	ref, err := collector.ParseRepoRef(cfg.Source.Type, repo)
	if err != nil {
		return err
	}
	col, err := collector.New(cfg, store, ref)
	if err != nil {
		return err
	}
	fmt.Printf("Collecting %s via %s provider...\n", repo, cfg.Source.Type)
	if err := col.Collect(ctx, ref); err != nil {
		return err
	}
	fmt.Println("Collection complete.")
	return nil
}

func collectSearch(ctx context.Context, cfg *config.Config, store *sqlite.Storage, opts domain.SearchOptions) error {
	prov, err := provider.Get(cfg.Source.Type, cfg.ProviderConfig())
	if err != nil {
		return err
	}
	searcher, ok := prov.(provider.Searcher)
	if !ok {
		return fmt.Errorf("provider %q does not support repository search", cfg.Source.Type)
	}

	query := collector.BuildSearchQuery(opts)
	fmt.Printf("Searching: %q\n", query)

	paths, err := dataset.EnsureDir(cfg.Workspace.DatasetsDir, collectFlags.name, collectFlags.keywords)
	if err != nil {
		return fmt.Errorf("create dataset dir: %w", err)
	}
	fmt.Printf("Dataset: %s (dir: %s)\n", paths.Name, paths.Dir)

	repos, err := searcher.SearchRepositories(ctx, opts)
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}
	if len(repos) == 0 {
		fmt.Println("No repositories found.")
		return nil
	}

	limit := collectFlags.limit
	if limit <= 0 || limit > len(repos) {
		limit = len(repos)
	}

	fmt.Printf("Found %d repositories (showing %d):\n", len(repos), limit)
	for i, repo := range repos[:limit] {
		fmt.Printf("  %d. %s (%d stars, %s)\n", i+1, repo.Ref.FullName, repo.Stars, repo.Language)
	}

	if collectFlags.dryRun {
		fmt.Println("Dry run — no data collected.")
		return nil
	}

	var failed int
	for i, repo := range repos[:limit] {
		fmt.Printf("\n[%d/%d] Collecting %s...\n", i+1, limit, repo.Ref.FullName)
		col, err := collector.New(cfg, store, repo.Ref)
		if err != nil {
			fmt.Printf("  skip: %v\n", err)
			failed++
			continue
		}
		if err := col.Collect(ctx, repo.Ref); err != nil {
			fmt.Printf("  failed: %v\n", err)
			failed++
			continue
		}
	}

	fmt.Printf("\nSearch collection complete. %d succeeded, %d failed.\n", limit-failed, failed)
	return nil
}

func buildSearchOptions() domain.SearchOptions {
	opts := domain.SearchOptions{
		Query:         collectFlags.query,
		Keywords:      collectFlags.keywords,
		Language:      collectFlags.language,
		CreatedAfter:  collectFlags.createdAfter,
		CreatedBefore: collectFlags.createdBefore,
		MinStars:      collectFlags.minStars,
		MaxStars:      collectFlags.maxStars,
		Topic:         collectFlags.topic,
		User:          collectFlags.user,
		Org:           collectFlags.org,
		Sort:          collectFlags.sort,
		Order:         collectFlags.order,
		ListOptions: domain.ListOptions{
			PerPage: collectFlags.limit,
		},
	}
	if collectFlags.forks != "" {
		if v, err := strconv.ParseBool(collectFlags.forks); err == nil {
			opts.Forks = &v
		}
	}
	if collectFlags.archived != "" {
		if v, err := strconv.ParseBool(collectFlags.archived); err == nil {
			opts.Archived = &v
		}
	}
	return opts
}
