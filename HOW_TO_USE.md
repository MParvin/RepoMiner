# HOW TO USE — RepoMiner (dataset-builder)

Zero-to-hero guide: install the tool, collect repository data, export a training dataset, optionally refine it with a local LLM, then fine-tune a model with **Unsloth AI**.

## Table of Contents

1. [What is RepoMiner?](#1-what-is-repominer)
2. [Prerequisites](#2-prerequisites)
3. [Installation and First Run](#3-installation-and-first-run)
4. [Configuration](#4-configuration)
5. [Primary Tutorial — Collect Cobra / Go Repos](#5-primary-tutorial--collect-cobra--go-repos)
6. [Generate Training Data](#6-generate-training-data)
7. [Dataset Format Reference](#7-dataset-format-reference)
8. [Optional — LLM Refinement](#8-optional--llm-refinement)
9. [CLI Command Reference](#9-cli-command-reference)
10. [Fine-Tuning with Unsloth AI](#10-fine-tuning-with-unsloth-ai)
11. [Advanced — Docker API](#11-advanced--docker-api)
12. [Troubleshooting](#12-troubleshooting)
13. [End-to-End Cheat Sheet](#13-end-to-end-cheat-sheet)

---

## 1. What is RepoMiner?

**RepoMiner** ships as the `dataset-builder` CLI. It collects repository metadata and activity (issues, pull requests, commits, branches, contributors) from source-control platforms, stores everything in a local SQLite database, and exports **instruction-tuning JSONL** (or a Hugging Face layout) ready for fine-tuning.

Mental model — three stages:

| Stage | Command | What you get |
|-------|---------|--------------|
| Collect | `collect` | Data in SQLite (`data/dataset-builder.db`). **No training file yet.** |
| Generate | `generate` | `datasets/<name>/dataset.jsonl` (or Hugging Face `train.jsonl`) |
| Refine (optional) | `refine` | Quality-filtered `refined.jsonl` via a local LLM |

```
GitHub / GitLab / local git
        │
        ▼
   collect  ──►  SQLite
        │
        ▼
   generate ──►  datasets/<name>/dataset.jsonl
        │
        ▼
   refine   ──►  refined.jsonl   (optional)
        │
        ▼
   Unsloth / Hugging Face / Ollama training
```

---

## 2. Prerequisites

| Requirement | Purpose |
|-------------|---------|
| **Go 1.26+** | Build the CLI |
| **git** | Local git provider and cloning |
| **GitHub or GitLab API token** | Authenticated collection (higher rate limits) |
| **NVIDIA GPU ≥ 8 GB VRAM** (16 GB recommended) | Unsloth fine-tuning only |
| **Python 3.10+** | Unsloth fine-tuning only |
| **Ollama** (optional) | `refine` command only |
| **sqlite3** CLI (optional) | Multi-repo merge helper in §6 |

---

## 3. Installation and First Run

```bash
git clone https://github.com/mparvin/repo-miner.git
cd repo-miner

# Build the binary → ./dataset-builder
make build

# Create workspace: config.yaml, data/, datasets/, repos/, SQLite DB
make init
```

What `make init` does:

- Writes `config.yaml` from the embedded sample (includes an `llm:` section)
- Creates `data/`, `datasets/`, `repos/`
- Migrates the SQLite schema at `data/dataset-builder.db`

Verify:

```bash
./dataset-builder version
./dataset-builder --help
```

---

## 4. Configuration

Edit `config.yaml` and set a token before collecting:

```yaml
source:
  type: github                 # github | gitlab | localgit
  url: https://api.github.com
  token: "ghp_YOUR_TOKEN_HERE" # required for reliable search/collection

analyzer:
  language: golang

storage:
  driver: sqlite
  path: data/dataset-builder.db

workspace:
  data_dir: data
  datasets_dir: datasets
  repos_dir: repos

queue:
  driver: memory               # memory | redis (redis needed for Docker API)

ranking:
  weights:
    agents_md: 30
    claude_md: 25
    cursor_dir: 15
    ai_commits: 10
    tests: 20
    ci: 15
    readme: 10
    docs: 10
    activity: 15
    maintainers: 10

llm:
  type: ollama
  base_url: http://localhost:11434
  model: llama3.2:latest
  threshold: 6.0
```

### Switching providers

**GitLab** (cloud or self-hosted):

```yaml
source:
  type: gitlab
  url: https://gitlab.com/api/v4   # or your instance API base
  token: "glpat-..."
```

**Local git** — clone into `repos/`, then collect:

```yaml
source:
  type: localgit
```

```bash
# Example: clone into repos/my-project
git clone https://github.com/spf13/cobra.git repos/my-project
./dataset-builder collect --repo local/my-project --name cobra-local
```

### Known limitations

- **Gitea** — provider is registered but not implemented yet
- **Gerrit** — mentioned in the README; no plugin yet
- Override config path with `-c` / `--config` on any command

---

## 5. Primary Tutorial — Collect Cobra / Go Repos

Goal: search GitHub for Go repositories related to **cobra**, created after 2026-01-01, collect up to 30 of them, then turn that data into a training set.

### 5.1 Dry-run first (recommended)

List matches without writing to the database:

```bash
./dataset-builder collect \
  --keywords cobra \
  --language Go \
  --created-after 2026-01-01 \
  --limit 30 \
  --dry-run
```

### 5.2 Collect for real

```bash
./dataset-builder collect \
  --keywords cobra \
  --language Go \
  --created-after 2026-01-01 \
  --limit 30 \
  --name cobra
```

What this does:

1. Builds a GitHub-style search (`cobra language:Go created:>2026-01-01`, sorted by stars)
2. Creates `datasets/cobra/`
3. For each of up to 30 repos, fetches and stores repository metadata, commits, branches, PRs, issues, and contributors (up to 100 of each list per repo) into SQLite
4. Prints progress like `[3/30] Collecting owner/repo...`

`--name cobra` sets the dataset directory name used later by `generate` and `refine`. If omitted, the name falls back to `--keywords`, then a random `dataset-<hex>` id.

### 5.3 Rank collected repos (pick the best ones)

```bash
./dataset-builder rank
```

Scores every collected repository on AI-related signals (`AGENTS.md`, `CLAUDE.md`, `.cursor/`, AI-looking commits) and engineering quality (tests, CI, README, docs, activity). Use the top-scoring repos as inputs to `generate`.

---

## 6. Generate Training Data

> **Important:** `generate` requires `--repo` and processes **one repository at a time**. Each write **overwrites** the target JSONL file (it does not append). For a multi-repo search, pick one repo or merge files yourself (see below).

### 6.A Single best repo (simplest)

After ranking, generate from a strong match (example: `spf13/cobra` if it was collected; otherwise use a full name from the `rank` output):

```bash
./dataset-builder generate --repo spf13/cobra --name cobra --format jsonl
# → datasets/cobra/dataset.jsonl
```

Hugging Face layout:

```bash
./dataset-builder generate --repo spf13/cobra --name cobra --format huggingface
# → datasets/cobra/train.jsonl
# → datasets/cobra/dataset_info.json
```

### 6.B Multi-repo merge (all collected cobra search results)

`generate` overwrites, so generate per repo into a temp file and concatenate:

```bash
mkdir -p datasets/cobra
> datasets/cobra/dataset.jsonl

for repo in $(sqlite3 data/dataset-builder.db \
  "SELECT full_name FROM repositories ORDER BY stars DESC LIMIT 30"); do
  echo "Generating $repo ..."
  ./dataset-builder generate --repo "$repo" --output /tmp/repominer-sample.jsonl \
    || { echo "skip $repo"; continue; }
  cat /tmp/repominer-sample.jsonl >> datasets/cobra/dataset.jsonl
done

wc -l datasets/cobra/dataset.jsonl
```

Python alternative (same idea):

```python
import json, sqlite3, subprocess, pathlib

out = pathlib.Path("datasets/cobra/dataset.jsonl")
out.parent.mkdir(parents=True, exist_ok=True)
out.write_text("")

conn = sqlite3.connect("data/dataset-builder.db")
repos = [r[0] for r in conn.execute(
    "SELECT full_name FROM repositories ORDER BY stars DESC LIMIT 30"
)]

with out.open("a") as f:
    for repo in repos:
        tmp = "/tmp/repominer-sample.jsonl"
        r = subprocess.run(
            ["./dataset-builder", "generate", "--repo", repo, "--output", tmp],
            capture_output=True, text=True,
        )
        if r.returncode != 0:
            print("skip", repo, r.stderr.strip())
            continue
        f.write(pathlib.Path(tmp).read_text())
        print("ok", repo)
```

---

## 7. Dataset Format Reference

Each line of the JSONL file is one `DatasetSample`:

| Field | Typical source | Description |
|-------|----------------|-------------|
| `instruction` | Issue title / PR title / commit subject | Primary task text |
| `context` | Issue body / PR description / commit metadata | Background |
| `solution` | PR merge info / full commit message | Target output (when present) |
| `metadata` | Provenance | `source`, `repo`, `author`, `number` or `hash`, `state` |

Example records:

```json
{"instruction":"perf: add fast paths for cleanPath","context":"## Summary\nThis PR adds fast paths...","solution":"PR #4735: open branch perf-path-clean-fast-path -> master","metadata":{"author":"james-yusuke","number":"4735","repo":"gin-gonic/gin","source":"pull_request","state":"open"}}
```

```json
{"instruction":"docs: fix BindXML comment","context":"Commit 34dac20 by greymoth","solution":"docs: fix BindXML comment referencing nonexistent binding.BindXML (#4717)","metadata":{"author":"greymoth","hash":"34dac20...","repo":"gin-gonic/gin","source":"commit"}}
```

```json
{"instruction":"Allow configuring the binding validator with Options","context":"### Feature Description\nThere is currently no first-class way...","metadata":{"author":"Hoffs","number":"4733","repo":"gin-gonic/gin","source":"issue","state":"open"}}
```

### Quality filters (applied during `generate`)

- Instruction length ≥ **10** characters
- Combined instruction + context + solution ≤ **8000** characters
- Must contain at least one alphabetic character
- HTML stripped, whitespace normalized
- Deduplicated by hash of normalized `instruction|solution`

### Dataset directory layout

```
datasets/cobra/
├── dataset.jsonl          # default --format jsonl
├── train.jsonl            # --format huggingface
├── dataset_info.json      # --format huggingface metadata
├── refined.jsonl          # after refine
├── refined-report.json    # per-sample scores / actions
└── manifest.json          # refine run metadata
```

Naming priority for `--name`: explicit `--name` → sanitized `--keywords` → random `dataset-<hex>`.

---

## 8. Optional — LLM Refinement

`refine` sends each sample to a local LLM (default: Ollama), scores it, and keeps / improves / rejects based on `llm.threshold` (default **6.0**).

### Configure

In `config.yaml`:

```yaml
llm:
  type: ollama
  base_url: http://localhost:11434
  model: llama3.2:latest
  threshold: 6.0
```

### Run Ollama and refine

```bash
ollama pull llama3.2

# Refine everything under datasets/cobra/dataset.jsonl
./dataset-builder refine --name cobra

# Smoke-test on the first 50 samples
./dataset-builder refine --name cobra --limit 50

# Or pass an explicit file
./dataset-builder refine --input datasets/cobra/dataset.jsonl --name cobra
```

Output:

```
Refinement complete:
  Total:    N
  Kept:     ...
  Improved: ...
  Rejected: ...
  Output:   datasets/cobra/refined.jsonl
```

Use `datasets/cobra/refined.jsonl` as the Unsloth training input when you refine.

Scoring dimensions: technical correctness, code quality, instruction clarity, solution validity, best practices, overall (0–10). Actions: `keep`, `improve`, `reject`.

---

## 9. CLI Command Reference

Global flag on every command:

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--config` | `-c` | `config.yaml` | Path to config file |

### `init`

Initialize workspace (config, folders, SQLite migration). No extra flags.

```bash
./dataset-builder init
# or: make init
```

### `collect`

Collect one repo or search-and-collect many.

| Flag | Default | Description |
|------|---------|-------------|
| `--repo` | | Single repo `owner/name` |
| `--query` | | Raw GitHub-style search query (overrides composed query) |
| `--keywords` | | Free-text search terms |
| `-l, --language` | | Language filter (e.g. `Go`) |
| `--created-after` | | `YYYY-MM-DD` |
| `--created-before` | | `YYYY-MM-DD` |
| `--min-stars` | `0` | Minimum stars |
| `--max-stars` | `0` | Maximum stars (`0` = no max) |
| `--topic` | | Topic filter |
| `--user` | | User filter |
| `--org` | | Organization filter |
| `--forks` | | Include forks: `true` / `false` |
| `--archived` | | Include archived: `true` / `false` |
| `--sort` | `stars` | `stars`, `updated`, `forks` |
| `--order` | `desc` | `desc`, `asc` |
| `--limit` | `30` | Max repos from search |
| `--dry-run` | `false` | List matches without collecting |
| `--name` | | Dataset directory name |

Requires `--repo` **or** at least one search flag.

### `generate`

| Flag | Default | Description |
|------|---------|-------------|
| `--repo` | **required** | `owner/name` to generate from |
| `--name` | | Dataset directory name |
| `--keywords` | | Name from keywords if `--name` omitted |
| `--output` | | Override output file/directory |
| `--format` | `jsonl` | `jsonl` or `huggingface` / `hf` |

### `refine`

| Flag | Default | Description |
|------|---------|-------------|
| `--input` | | Input JSONL path |
| `--name` | | Dataset name under `datasets/<name>/` |
| `--keywords` | | Name from keywords if `--name` omitted |
| `--output` | | Override refined JSONL path |
| `--limit` | `0` | Max samples (`0` = all) |

Requires `--input`, `--name`, or `--keywords`.

### `rank`

Rank all collected repositories. No flags.

```bash
./dataset-builder rank
```

### `analyze [path]`

Analyze Go source at a path (AST). Prints JSON to stdout. Results are **not** fed into `generate` today.

```bash
./dataset-builder analyze ./repos/my-clone
```

### `version` / `completion`

```bash
./dataset-builder version
./dataset-builder completion bash   # also: zsh, fish, powershell
```

### More collect examples

```bash
# Single known repo
./dataset-builder collect --repo gin-gonic/gin --name gin

# Popular Go repos, sorted by recent updates
./dataset-builder collect --language Go --min-stars 100 --sort updated --limit 20 --name go-popular

# Raw query
./dataset-builder collect --query "topic:cli language:Go stars:>50" --limit 15 --name go-cli
```

---

## 10. Fine-Tuning with Unsloth AI

This section fine-tunes **Qwen3-8B** on a RepoMiner JSONL export using **Unsloth** (roughly 2× faster training with up to ~70% less memory).

### Field mapping

| RepoMiner field | Training role |
|-----------------|---------------|
| `instruction` | Task / instruction |
| `context` | Input / background |
| `solution` | Target response (when present) |

### 10.1 Prerequisites and install

- NVIDIA GPU with ≥ **8 GB** VRAM (16 GB recommended)
- Python **3.10+**

```bash
# Local Linux / Windows
pip install unsloth

# Google Colab
pip install "unsloth[colab-new] @ git+https://github.com/unslothai/unsloth.git"
```

Also ensure Hugging Face datasets is available:

```bash
pip install datasets trl transformers
```

### 10.2 Load Qwen3-8B (4-bit)

```python
from unsloth import FastLanguageModel
import torch

model, tokenizer = FastLanguageModel.from_pretrained(
    model_name = "unsloth/Qwen3-8B-bnb-4bit",
    max_seq_length = 2048,
    dtype = None,          # auto: bfloat16 on Ampere+, else float16
    load_in_4bit = True,   # ~5–6 GB instead of ~14 GB
)
```

### 10.3 Add LoRA adapters

```python
model = FastLanguageModel.get_peft_model(
    model,
    r = 16,
    target_modules = ["q_proj", "k_proj", "v_proj", "o_proj",
                      "gate_proj", "up_proj", "down_proj"],
    lora_alpha = 16,
    lora_dropout = 0,
    bias = "none",
    use_gradient_checkpointing = "unsloth",
)
```

### 10.4 Load your RepoMiner dataset

```python
from datasets import load_dataset
from unsloth.chat_templates import get_chat_template

# Default JSONL from generate
dataset = load_dataset("json", data_files="datasets/cobra/dataset.jsonl", split="train")

# Or after refine:
# dataset = load_dataset("json", data_files="datasets/cobra/refined.jsonl", split="train")

# Or Hugging Face layout:
# dataset = load_dataset("json", data_files="datasets/cobra/train.jsonl", split="train")

tokenizer = get_chat_template(
    tokenizer,
    chat_template = "qwen-2.5",
)
```

Optional — keep solution-rich samples (PRs and commits) only:

```python
dataset = dataset.filter(
    lambda x: (x.get("metadata") or {}).get("source") in ("pull_request", "commit")
)
```

### 10.5 Format prompts (Alpaca-style)

Issues often have no `solution`; the formatter falls back to `context` so every row still produces a `text` field. Prefer `refine` or the PR/commit filter above for higher-quality pairs.

```python
def formatting_prompts_func(examples):
    instructions = examples["instruction"]
    contexts = examples.get("context") or [""] * len(instructions)
    solutions = examples.get("solution") or [""] * len(instructions)
    texts = []
    for instruction, context, solution in zip(instructions, contexts, solutions):
        input_text = context or ""
        output_text = solution or context or ""
        text = (
            f"### Instruction:\n{instruction}\n\n"
            f"### Input:\n{input_text}\n\n"
            f"### Response:\n{output_text}{tokenizer.eos_token}"
        )
        texts.append(text)
    return {"text": texts}

dataset = dataset.map(formatting_prompts_func, batched=True)
```

### 10.6 Train with SFTTrainer

```python
from trl import SFTTrainer
from transformers import TrainingArguments

trainer = SFTTrainer(
    model = model,
    tokenizer = tokenizer,
    train_dataset = dataset,
    dataset_text_field = "text",
    max_seq_length = 2048,
    args = TrainingArguments(
        per_device_train_batch_size = 2,
        gradient_accumulation_steps = 4,   # effective batch size = 8
        warmup_steps = 5,
        max_steps = 60,                    # quick smoke test; for real runs use num_train_epochs=1
        # num_train_epochs = 1,
        learning_rate = 2e-4,
        fp16 = not torch.cuda.is_bf16_supported(),
        bf16 = torch.cuda.is_bf16_supported(),
        logging_steps = 1,
        optim = "adamw_8bit",
        output_dir = "outputs_cobra",
    ),
)

trainer.train()
```

### 10.7 Inference and save (GGUF / Ollama)

```python
FastLanguageModel.for_inference(model)

prompt = (
    "### Instruction:\nExplain how to add a persistent flag in Cobra.\n\n"
    "### Input:\n\n\n"
    "### Response:\n"
)
inputs = tokenizer([prompt], return_tensors="pt").to("cuda")
outputs = model.generate(**inputs, max_new_tokens=128)
print(tokenizer.batch_decode(outputs)[0])

# Save adapters / full weights, then GGUF for Ollama or llama.cpp
model.save_pretrained("qwen3_cobra_lora")
tokenizer.save_pretrained("qwen3_cobra_lora")
model.save_pretrained_gguf(
    "qwen3_cobra_gguf",
    tokenizer,
    quantization_method = "q4_k_m",
)
```

### 10.8 Complete script (`train_cobra.py`)

Save next to the project root (or adjust paths), then run `python train_cobra.py`:

```python
"""Fine-tune Qwen3-8B on a RepoMiner cobra dataset with Unsloth."""

from unsloth import FastLanguageModel
from unsloth.chat_templates import get_chat_template
from datasets import load_dataset
from trl import SFTTrainer
from transformers import TrainingArguments
import torch

DATA_FILE = "datasets/cobra/dataset.jsonl"  # or refined.jsonl / train.jsonl
MAX_SEQ = 2048

model, tokenizer = FastLanguageModel.from_pretrained(
    model_name="unsloth/Qwen3-8B-bnb-4bit",
    max_seq_length=MAX_SEQ,
    dtype=None,
    load_in_4bit=True,
)

model = FastLanguageModel.get_peft_model(
    model,
    r=16,
    target_modules=["q_proj", "k_proj", "v_proj", "o_proj",
                    "gate_proj", "up_proj", "down_proj"],
    lora_alpha=16,
    lora_dropout=0,
    bias="none",
    use_gradient_checkpointing="unsloth",
)

tokenizer = get_chat_template(tokenizer, chat_template="qwen-2.5")

dataset = load_dataset("json", data_files=DATA_FILE, split="train")
# Optional quality filter:
# dataset = dataset.filter(
#     lambda x: (x.get("metadata") or {}).get("source") in ("pull_request", "commit")
# )


def formatting_prompts_func(examples):
    instructions = examples["instruction"]
    contexts = examples.get("context") or [""] * len(instructions)
    solutions = examples.get("solution") or [""] * len(instructions)
    texts = []
    for instruction, context, solution in zip(instructions, contexts, solutions):
        input_text = context or ""
        output_text = solution or context or ""
        texts.append(
            f"### Instruction:\n{instruction}\n\n"
            f"### Input:\n{input_text}\n\n"
            f"### Response:\n{output_text}{tokenizer.eos_token}"
        )
    return {"text": texts}


dataset = dataset.map(formatting_prompts_func, batched=True)

trainer = SFTTrainer(
    model=model,
    tokenizer=tokenizer,
    train_dataset=dataset,
    dataset_text_field="text",
    max_seq_length=MAX_SEQ,
    args=TrainingArguments(
        per_device_train_batch_size=2,
        gradient_accumulation_steps=4,
        warmup_steps=5,
        max_steps=60,
        learning_rate=2e-4,
        fp16=not torch.cuda.is_bf16_supported(),
        bf16=torch.cuda.is_bf16_supported(),
        logging_steps=1,
        optim="adamw_8bit",
        output_dir="outputs_cobra",
    ),
)

trainer.train()

FastLanguageModel.for_inference(model)
prompt = (
    "### Instruction:\nHow do I define a Cobra subcommand?\n\n"
    "### Input:\n\n\n### Response:\n"
)
inputs = tokenizer([prompt], return_tensors="pt").to("cuda")
print(tokenizer.batch_decode(model.generate(**inputs, max_new_tokens=128))[0])

model.save_pretrained("qwen3_cobra_lora")
tokenizer.save_pretrained("qwen3_cobra_lora")
model.save_pretrained_gguf("qwen3_cobra_gguf", tokenizer, quantization_method="q4_k_m")
```

---

## 11. Advanced — Docker API

For distributed collection via Redis + API + worker:

```bash
docker compose up -d --build
```

Services: Redis, `dataset-api` on host port **8081**, and a worker.

`docker-compose.yml` sets `API_KEY` for the API service. Export the same value locally before calling curl:

```bash
export API_KEY="$(grep -E '^\s+- API_KEY=' docker-compose.yml | cut -d= -f2)"

# Queue a collect job
curl -X POST http://localhost:8081/jobs/collect \
  -H "X-API-Key: ${API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{"repo":"spf13/cobra"}'

# Health (no auth)
curl http://localhost:8081/health

# Job status (auth required) — replace {id} with the returned job id
curl http://localhost:8081/jobs/{id}/status \
  -H "X-API-Key: ${API_KEY}"
```

Notes:

- The HTTP API currently queues **`collect`** jobs only
- `generate` / `refine` / Unsloth remain CLI (or local Python) workflows
- Job status is in-memory on the API process (lost on restart)

---

## 12. Troubleshooting

| Problem | Fix |
|---------|-----|
| `specify --repo or search flags` | Pass `--repo` or at least one of `--query`, `--keywords`, `--language`, dates, etc. |
| HTTP 401 / rate limit | Set `source.token` in `config.yaml` |
| `no samples generated for ...` | That repo has no usable issues/PRs/commits in SQLite — re-collect or pick another repo |
| Second `generate` wiped the first | Expected overwrite — use the multi-repo merge loop in §6.B or different `--output` paths |
| `refine` cannot reach the model | Start Ollama (`ollama serve`), `ollama pull <model>`, match `llm.model` in config |
| Unsloth CUDA OOM | Lower `per_device_train_batch_size`, keep `load_in_4bit=True`, reduce `max_seq_length` |
| Provider `gitea` / Gerrit errors | Not implemented yet — use `github`, `gitlab`, or `localgit` |
| Empty search results | Relax filters (`--created-after`, `--min-stars`) or try `--dry-run` to inspect the query |

---

## 13. End-to-End Cheat Sheet

```bash
# 1. Build & init
make build && make init
# Edit config.yaml → set source.token

# 2. Preview search
./dataset-builder collect \
  --keywords cobra --language Go --created-after 2026-01-01 --limit 30 --dry-run

# 3. Collect
./dataset-builder collect \
  --keywords cobra --language Go --created-after 2026-01-01 --limit 30 --name cobra

# 4. Rank and generate (single top repo)
./dataset-builder rank
./dataset-builder generate --repo spf13/cobra --name cobra --format huggingface

# 5. Optional refine
./dataset-builder refine --name cobra --limit 100

# 6. Fine-tune with Unsloth
python train_cobra.py
```

For architecture details see [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md). For a short project overview see [README.md](README.md).
