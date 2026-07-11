# RepoMiner — Software Engineering Dataset Builder

A plugin-based system for collecting, analyzing, and generating software engineering datasets from multiple source control platforms.

## Supported Sources (planned)

- GitHub
- GitLab Cloud / Self-Hosted
- Gitea
- Gerrit
- Local Git repositories

## Quick Start

```bash
# Build
make build

# Initialize workspace (config, folders, database)
make init

# Show CLI help
make help
```

## Project Structure

```
dataset-builder
├── cmd/                    # CLI entrypoint
├── config/                 # Example configuration
├── internal/
│   ├── analyzer/           # Language analyzer registry
│   ├── collector/          # Repository collection orchestrator
│   ├── config/             # Configuration loading
│   ├── core/
│   │   ├── domain/         # Domain models
│   │   ├── pipeline/       # Dataset pipeline
│   │   ├── provider/       # Repository provider registry
│   │   └── queue/          # Job queue
│   ├── generator/          # Dataset generation orchestrator
│   └── storage/            # Storage interface + SQLite
└── plugins/
    ├── analyzers/          # Language analyzer plugins
    └── providers/          # Repository provider plugins
```

## Configuration

Copy the example config and customize:

```bash
cp config/config.example.yaml config.yaml
```

Set `source.type` to select a repository provider and `analyzer.language` to select a language analyzer — no code changes required.

## Development

```bash
make test    # Run tests
make vet     # Static analysis
```

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for design decisions.
