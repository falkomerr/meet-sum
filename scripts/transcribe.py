#!/usr/bin/env python3
"""Transcribe an audio file using OpenAI Whisper."""

import argparse
import json
import ssl
import sys

# On macOS, Python may not trust the system certificate store, causing SSL errors
# when downloading Whisper model weights. This patches the default context to allow it.
ssl._create_default_https_context = ssl._create_unverified_context


def parse_args():
    parser = argparse.ArgumentParser(description="Transcribe audio using Whisper")
    parser.add_argument("--model", required=True, help="Whisper model name (e.g. base, small, medium, large)")
    parser.add_argument("--input", required=True, help="Path to input audio file")
    parser.add_argument("--output", required=True, help="Path to write JSON output")
    return parser.parse_args()


def main():
    args = parse_args()

    print(f"Loading Whisper model '{args.model}'...", file=sys.stderr)
    try:
        import whisper
    except ImportError:
        print("ERROR: openai-whisper is not installed. Run: pip install openai-whisper", file=sys.stderr)
        sys.exit(1)

    try:
        model = whisper.load_model(args.model)
    except Exception as e:
        print(f"ERROR: Failed to load model '{args.model}': {e}", file=sys.stderr)
        sys.exit(1)

    print(f"Transcribing '{args.input}'...", file=sys.stderr)
    try:
        result = model.transcribe(args.input)
    except FileNotFoundError:
        print(f"ERROR: Input file not found: {args.input}", file=sys.stderr)
        sys.exit(1)
    except Exception as e:
        print(f"ERROR: Transcription failed: {e}", file=sys.stderr)
        sys.exit(1)

    # Extract duration from segments if available
    duration = 0.0
    segments = result.get("segments", [])
    if segments:
        duration = segments[-1].get("end", 0.0)

    output = {
        "text": result.get("text", "").strip(),
        "language": result.get("language", ""),
        "duration": duration,
    }

    print(f"Writing output to '{args.output}'...", file=sys.stderr)
    try:
        with open(args.output, "w", encoding="utf-8") as f:
            json.dump(output, f, ensure_ascii=False, indent=2)
    except Exception as e:
        print(f"ERROR: Failed to write output: {e}", file=sys.stderr)
        sys.exit(1)

    print("Done.", file=sys.stderr)


if __name__ == "__main__":
    main()
