# meet-sum

> Record, transcribe, and summarize your meetings locally — no cloud, no subscriptions.

[![CI](https://github.com/falkomerr/meet-sum/actions/workflows/ci.yml/badge.svg)](https://github.com/falkomerr/meet-sum/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/falkomerr/meet-sum)](https://github.com/falkomerr/meet-sum/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](go.mod)

---

## How It Works

```
┌─────────┐    ┌───────────┐    ┌──────────┐    ┌──────────────┐
│  Record │───▶│  Whisper  │───▶│  Ollama  │───▶│  Markdown    │
│ (ffmpeg)│    │(local STT)│    │(local LLM│    │   Summary    │
└─────────┘    └───────────┘    └──────────┘    └──────────────┘
```

Everything runs on your machine. No data leaves your network.

---

## Features

- 🎙️ Records system audio + microphone simultaneously via ffmpeg
- 🔤 Transcribes with [Whisper](https://github.com/openai/whisper) — runs fully offline
- 🤖 Summarizes with [Ollama](https://ollama.ai) — local LLM, no API keys needed
- 💾 SQLite transcription cache — re-summarize with different prompts without re-transcribing
- 📦 Compresses recordings with zstd after processing
- 🖥️ Works on macOS, Linux, and Windows

---

## Quick Install

```bash
curl -fsSL https://raw.githubusercontent.com/falkomerr/meet-sum/main/scripts/install.sh | bash
```

Installs the binary and checks for required dependencies.

---

## Prerequisites

| Tool | Purpose | Install |
|------|---------|---------|
| ffmpeg | Audio recording & extraction | `brew install ffmpeg` / `apt install ffmpeg` |
| Python 3.8+ | Runs Whisper | `brew install python` / `apt install python3` |
| openai-whisper | Speech-to-text | `pip install openai-whisper` |
| Ollama | Local LLM for summarization | [ollama.ai](https://ollama.ai) |
| BlackHole 2ch *(macOS only)* | System audio capture | [existential.audio/blackhole](https://existential.audio/blackhole/) |

> **Linux:** Uses PulseAudio monitor source automatically — no extra setup needed.
> **Windows:** Uses WASAPI loopback automatically — no extra setup needed.

---

## Build from Source

```bash
git clone https://github.com/falkomerr/meet-sum
cd meet-sum
CGO_ENABLED=1 go build -o meet-sum .
sudo mv meet-sum /usr/local/bin/
pip install openai-whisper
```

---

## Usage

### 1. First-time setup

```bash
meet-sum init
```

An interactive wizard configures your Whisper model, Ollama model, and output directories. Configuration is saved to `~/.config/meet-sum/config.yaml`.

### 2. Record a meeting

```bash
meet-sum record
# Press Enter to stop recording
```

```
Recording... Press Enter to stop.
[Enter]
Recording saved to: ./video_dist/recording_2024-01-15_10-30-00.wav
```

Captures system audio + microphone simultaneously. Supports macOS, Linux, and Windows.

### 3. Transcribe & summarize

```bash
meet-sum summarize ./video_dist/recording_2024-01-15_10-30-00.wav
```

Full pipeline:
1. Extract audio (if video file)
2. Check transcription cache (SQLite, keyed by file hash + Whisper model)
3. Transcribe with Whisper
4. Summarize with Ollama
5. Save Markdown to `transcribed_dist/<name>.md`
6. Compress original file with zstd

**Supported formats:**
- Audio: `.wav`, `.mp3`, `.ogg`, `.flac`
- Video: `.mp4`, `.mkv`, `.webm`, `.avi`

**Re-summarize with a custom prompt** (uses cached transcription — no Whisper re-run):

```bash
meet-sum summarize recording.wav --prompt "Extract action items and deadlines only"
```

### 4. List recordings

```bash
meet-sum list
```

```
=== Recordings (./video_dist) ===
  recording_2024-01-15_10-30-00.wav        (45.2 MB)
  meeting_2024-01-14_09-00-00.wav.zst      (18.1 MB compressed)

=== Summaries (./transcribed_dist) ===
  recording_2024-01-15_10-30-00.md         (3.7 KB)
  meeting_2024-01-14_09-00-00.md           (4.1 KB)
```

### 5. Manage configuration

```bash
meet-sum config show
meet-sum config set whisper-model turbo
meet-sum config set api-model mistral:7b
meet-sum config set video-dir ~/meetings/recordings
meet-sum config set transcribed-dir ~/meetings/summaries
```

---

## Configuration

| Key | Description | Default |
|-----|-------------|---------|
| `whisper-model` | Whisper model size | `base` |
| `api-model` | Ollama model | `qwen2.5` |
| `api-base-url` | Ollama API endpoint | `http://localhost:11434/v1` |
| `video-dir` | Recordings output directory | `./video_dist` |
| `transcribed-dir` | Summaries output directory | `./transcribed_dist` |

---

## Whisper Model Guide

| Model | Size | Speed | Quality |
|-------|------|-------|---------|
| `tiny` | 39 MB | Very fast | Low |
| `base` | 74 MB | Fast | Fair |
| `small` | 244 MB | Medium | Good |
| `turbo` | 809 MB | Fast | Very good ★ |
| `medium` | 769 MB | Slower | Very good |
| `large` | 1.5 GB | Slow | Best |

### Recommended by hardware

| Hardware | Recommended | Why |
|----------|-------------|-----|
| MacBook Air M1/M2 (8 GB) | `turbo` or `small` | Neural Engine accelerates inference, 8 GB sufficient |
| MacBook Air M2/M3 (16 GB) | `turbo` or `medium` | More RAM → stable medium |
| MacBook Pro M1/M2 Pro/Max | `turbo` or `medium` | 16–32 GB, fast Neural Engine |
| MacBook Pro M2/M3 Max/Ultra | `medium` or `large` | 64–192 GB, best quality |
| MacBook Pro M4 Pro/Max | `large` | Fastest hardware, `large` flies |
| Intel Mac (2018–2021) | `small` or `base` | No Neural Engine, CPU inference |
| Intel Mac (2015–2017) | `base` or `tiny` | Limited resources, slow CPU |

> **Practical rule:** Start with `turbo` — it's the best balance for Apple Silicon. Switch to `large` only if you need maximum quality (heavy accents, technical jargon, quiet recordings).

---

## File Structure

```
~/.config/meet-sum/
  config.yaml       # configuration
  cache.db          # SQLite transcription cache

./video_dist/
  recording_*.wav       # original recordings
  recording_*.wav.zst   # compressed after summarization

./transcribed_dist/
  recording_*.md        # Markdown summaries
```

---

## Running Tests

```bash
CGO_ENABLED=1 go test ./...
```

---

## Contributing

Contributions are welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

---

## License

MIT — see [LICENSE](LICENSE)
