# Contributing to meet-sum

Thank you for your interest in contributing! meet-sum is an open source CLI tool for local meeting transcription and summarization, and we welcome contributions of all kinds.

---

## Ways to Contribute

- **Report bugs** — open an issue with the bug report template
- **Request features** — open an issue with the feature request template
- **Fix bugs** — submit a PR with a fix and a test
- **Add features** — discuss in an issue first, then submit a PR
- **Improve documentation** — fixes and clarifications are always welcome

---

## Development Setup

### Prerequisites

- Go 1.21+
- Python 3.8+
- ffmpeg
- CGO toolchain (gcc/clang) — required for go-sqlite3

### Clone and build

```bash
git clone https://github.com/falkomerr/meet-sum
cd meet-sum

# Build
CGO_ENABLED=1 go build -o meet-sum .

# Run tests
CGO_ENABLED=1 go test ./...
```

### Install Python dependencies (for Whisper integration tests)

```bash
pip install openai-whisper
```

---

## Project Structure

```
meet-sum/
├── main.go                     # Entry point
├── cmd/                        # Cobra CLI commands
│   ├── root.go                 # Root command, global flags
│   ├── init.go                 # Interactive setup wizard
│   ├── record.go               # Audio recording command
│   ├── summarize.go            # Full transcribe+summarize pipeline
│   ├── list.go                 # List recordings and summaries
│   └── config.go               # Config management commands
├── internal/
│   ├── config/                 # Configuration (Viper-based, YAML)
│   ├── recorder/               # ffmpeg audio recording, cross-platform device detection
│   ├── transcriber/            # Whisper integration via Python subprocess
│   ├── summarizer/             # Ollama/OpenAI-compatible API client
│   ├── cache/                  # SQLite transcription cache
│   └── compressor/             # zstd compression/decompression
└── scripts/
    ├── transcribe.py           # Python entry point for Whisper
    └── install.sh              # curl | bash installer
```

---

## Running Tests

```bash
# All tests
CGO_ENABLED=1 go test ./...

# Verbose output
CGO_ENABLED=1 go test -v ./...

# Specific package
CGO_ENABLED=1 go test -v ./internal/cache/...
CGO_ENABLED=1 go test -v ./internal/compressor/...

# With race detector
CGO_ENABLED=1 go test -race ./...
```

> Note: `CGO_ENABLED=1` is required because `go-sqlite3` uses CGO.

---

## Code Style

- Follow standard Go conventions — run `gofmt` before committing
- Keep functions small and focused
- Error wrapping: `fmt.Errorf("operation: %w", err)`
- No linter warnings — the CI runs `golangci-lint`

---

## Commit Messages

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add support for mp4 video files
fix: handle empty transcription gracefully
docs: update Whisper model selection guide
chore: update go dependencies
test: add cache eviction tests
```

---

## Pull Request Process

1. **Fork** the repository
2. **Create a branch** from `main`: `git checkout -b feat/my-feature`
3. **Make your changes** with tests
4. **Verify** everything passes: `CGO_ENABLED=1 go test ./...`
5. **Push** your branch and open a PR
6. **Fill out** the PR template
7. **Wait for review** — we aim to respond within a few days

---

## Adding a New Summarizer

The summarizer is pluggable via the `Summarizer` interface in `internal/summarizer/summarizer.go`:

```go
type Summarizer interface {
    Summarize(transcription string, prompt string) (string, error)
}
```

To add a new backend (e.g., Anthropic Claude):

1. Create `internal/summarizer/claude.go`
2. Implement the `Summarizer` interface
3. Add configuration fields in `internal/config/config.go`
4. Wire it up in `cmd/summarize.go`
5. Add tests in `internal/summarizer/claude_test.go`

---

## Reporting Bugs

Please use the [bug report template](.github/ISSUE_TEMPLATE/bug_report.md) and include:

- Your OS and architecture
- `meet-sum --version` output
- The full error message or unexpected behavior
- Steps to reproduce

---

## Feature Requests

Please use the [feature request template](.github/ISSUE_TEMPLATE/feature_request.md). Describe the problem you're trying to solve, not just the solution — this helps us find the best approach together.

---

## Questions?

Open a [GitHub Discussion](https://github.com/falkomerr/meet-sum/discussions) or an issue tagged `question`.
