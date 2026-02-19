# Meet-Summarizer: Epic Design Document

## Requirements (IMMUTABLE)

1. CLI tool `meet-sum` written in Go with Python subprocess for Whisper only
2. `meet-sum init` creates video_dist/, transcribed_dist/ directories and runs setup wizard
3. `meet-sum record` captures system audio + microphone via ffmpeg, saves to video_dist/
4. Recording starts/stops via Enter key in CLI (no global hotkey daemon)
5. Cross-platform audio recording: macOS (BlackHole + AVFoundation), Linux (PulseAudio), Windows (WASAPI/dshow)
6. `meet-sum summarize <file>` extracts audio → /tmp/audio_from_video/, transcribes via Whisper, summarizes via Ollama
7. Whisper model selectable during init or via `meet-sum config set whisper-model <model>`
8. Summarization via Ollama (local) through OpenAI-compatible Go SDK
9. Summarization prompt preserves ALL important details (names, dates, numbers, action items, decisions)
10. Output: Markdown file in transcribed_dist/<filename>.md
11. After summarization, compress original with zstd; install zstd automatically if missing
12. Cache: SQLite stores transcriptions keyed by file hash + whisper model
13. If file already summarized → use cached transcription (skip Whisper)
14. `--prompt` flag: re-summarize from cached transcription with custom prompt
15. Config stored at ~/.config/meet-sum/config.yaml (API key, whisper model, paths)
16. Summarizer interface must be pluggable (Ollama first, others later)

## Success Criteria (MUST ALL BE TRUE)

- [ ] `meet-sum init` creates directories and config file via interactive wizard
- [ ] `meet-sum record` captures system audio + mic, saves WAV to video_dist/
- [ ] `meet-sum summarize <file>` produces Markdown summary in transcribed_dist/
- [ ] Cached transcriptions are reused (no re-transcription for same file)
- [ ] `--prompt` flag re-summarizes from cache with custom instructions
- [ ] Original files compressed with zstd after summarization
- [ ] Config commands work: `meet-sum config set/show`
- [ ] Cross-platform: macOS, Linux, Windows
- [ ] All builds pass
- [ ] Pre-commit hooks passing (if configured)

## Anti-Patterns (FORBIDDEN)

- NO embedding Python logic in Go (Go calls Python subprocess ONLY for Whisper)
- NO hardcoded API keys (must come from config or env)
- NO in-memory caching (must persist in SQLite across sessions)
- NO re-transcribing already cached files (cache lookup is mandatory before Whisper)
- NO blocking UI during long operations without progress indication
- NO platform-specific Go code without build tags (use build tags for OS-specific recorder)
- NO storing config in project directory (must use ~/.config/meet-sum/)
- NO deleting original files without successful zstd compression verification

## Approach

Go CLI (cobra/viper) as the main binary. Python used exclusively for Whisper transcription
via subprocess (scripts/transcribe.py). Go handles everything else: recording via ffmpeg
subprocess, SQLite caching, zstd compression (native Go library), Ollama API calls via
OpenAI-compatible Go SDK, and config management via viper.

## Architecture

```
meet-sum (Go binary)
├── cmd/                       # CLI commands (cobra)
│   ├── root.go                # Root command, global flags
│   ├── init.go                # First-time setup wizard
│   ├── record.go              # Audio recording
│   ├── summarize.go           # Transcribe + summarize pipeline
│   └── config.go              # Config management
├── internal/
│   ├── recorder/
│   │   ├── recorder.go        # Recorder interface + ffmpeg logic
│   │   ├── devices_darwin.go  # macOS: BlackHole + AVFoundation
│   │   ├── devices_linux.go   # Linux: PulseAudio
│   │   └── devices_windows.go # Windows: WASAPI/dshow
│   ├── transcriber/
│   │   └── whisper.go         # Python subprocess wrapper
│   ├── summarizer/
│   │   ├── summarizer.go      # Summarizer interface
│   │   └── ollama.go          # Ollama local via OpenAI SDK
│   ├── cache/
│   │   └── sqlite.go          # SQLite transcription cache
│   ├── compressor/
│   │   └── zstd.go            # zstd compression
│   └── config/
│       └── config.go          # Viper-based config management
├── scripts/
│   └── transcribe.py          # Whisper transcription entry point
├── go.mod / go.sum
└── requirements.txt           # Python: openai-whisper
```

### Data Flow

```
RECORD:
  meet-sum record → ffmpeg (sys audio + mic) → video_dist/<timestamp>.wav

SUMMARIZE:
  meet-sum summarize <file>
  → cache.Get(fileHash, whisperModel)
  → MISS: extract audio → Python whisper → cache.Set() → transcription
  → HIT: transcription from cache
  → Ollama API (with default or --prompt) → summary
  → Save → transcribed_dist/<name>.md
  → zstd compress original → delete original

CONFIG:
  meet-sum config set whisper-model large → update YAML
  meet-sum config show → print current config
```

### Summarization Prompt

```
You are a professional meeting notes assistant. Analyze the following
meeting transcript and create a comprehensive summary.

## Structure:
1. **Meeting Overview**: 2-3 sentences — purpose and outcome
2. **Key Discussion Points**: All significant topics with details
3. **Decisions Made**: Every decision with context
4. **Action Items**: Tasks, responsible persons, deadlines
5. **Important Details**: Numbers, dates, names, technical terms
6. **Open Questions**: Unresolved issues for follow-up

## Rules:
- Do NOT omit any important details
- Preserve all names, dates, numbers, technical terms exactly
- Attribute statements to speakers when identifiable
- Use the original language of the transcript
- Structure with clear headers and bullet points
```

### Cross-Platform Recording

| OS | System Audio | Microphone | FFmpeg flags |
|----|-------------|------------|-------------|
| macOS | BlackHole 2ch | AVFoundation default | `-f avfoundation -i ":BlackHole 2ch" -f avfoundation -i ":default"` |
| Linux | PulseAudio monitor | PulseAudio default | `-f pulse -i <monitor> -f pulse -i default` |
| Windows | WASAPI loopback | dshow default | `-f dshow -i audio="<loopback>" -f dshow -i audio="<mic>"` |

### Key Libraries (Go)

- github.com/spf13/cobra — CLI framework
- github.com/spf13/viper — Config management
- github.com/sashabaranov/go-openai — OpenAI-compatible SDK for Ollama
- github.com/mattn/go-sqlite3 — SQLite driver
- github.com/klauspost/compress/zstd — zstd compression
- github.com/schollz/progressbar/v3 — Progress indication

### Key Libraries (Python)

- openai-whisper — Speech-to-text

## Design Rationale

### Problem
Meeting recordings pile up without summaries. Manually transcribing and summarizing
is time-consuming. Existing tools are either paid SaaS or don't integrate into CLI workflow.

### Approaches Considered

#### 1. Go + Python only for Whisper (CHOSEN)
Go handles CLI, recording, caching, compression, and Ollama API calls.
Python subprocess called only for Whisper transcription.

**Chosen because:** Minimizes Python surface area, single Go binary + one .py script,
Go has excellent OpenAI-compatible SDK for Ollama calls.

#### 2. Go CLI + Python for all ML (REJECTED)
Python handles both transcription and summarization.

**REJECTED BECAUSE:** Duplicates HTTP client logic, more Python code to maintain,
Go can natively call OpenAI-compatible APIs.

**DO NOT REVISIT UNLESS:** Ollama API becomes incompatible with OpenAI SDK.

#### 3. Go + Python FastAPI (REJECTED)
Python as local HTTP server, Go as client.

**REJECTED BECAUSE:** Overkill for local tool, lifecycle management complexity.

**DO NOT REVISIT UNLESS:** Need to support remote/distributed processing.

## Scope Boundaries

**In scope:**
- CLI tool with init, record, summarize, config commands
- Cross-platform audio recording (system + mic)
- Whisper transcription via Python subprocess
- Ollama summarization via Go OpenAI SDK
- SQLite caching of transcriptions
- zstd compression of originals
- Pluggable summarizer interface

**Out of scope:**
- GUI / web interface
- Real-time transcription during recording
- Speaker diarization (who said what)
- Video recording (audio only)
- Batch processing (one file at a time)
- Global hotkeys / daemon mode
