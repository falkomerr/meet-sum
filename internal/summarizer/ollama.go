package summarizer

import (
	"context"
	"fmt"
	"os"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

const (
	defaultBaseURL   = "http://localhost:11434/v1"
	defaultModel     = "mistral:7b"
	defaultMaxTokens = 4096
)

// OllamaSummarizer implements Summarizer using an OpenAI-compatible API (Ollama local).
type OllamaSummarizer struct {
	client *openai.Client
	model  string
}

// NewOllamaSummarizer creates a new OllamaSummarizer with the given API key, base URL, and model.
// If baseURL is empty, the default Ollama endpoint is used (http://localhost:11434/v1).
// If model is empty, qwen2.5 is used.
func NewOllamaSummarizer(apiKey, baseURL, model string) *OllamaSummarizer {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if model == "" {
		model = defaultModel
	}

	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = baseURL

	return &OllamaSummarizer{
		client: openai.NewClientWithConfig(cfg),
		model:  model,
	}
}

// Summarize sends the transcription to the local Ollama model using prompt as the system message,
// then runs a review pass to remove any invented content.
func (o *OllamaSummarizer) Summarize(transcription string, prompt string) (string, error) {
	// Step 1: generate initial summary.
	draft, err := o.complete(prompt, "Summarize this transcript in Russian:\n\n"+transcription)
	if err != nil {
		return "", fmt.Errorf("Ollama summarization request failed: %w", err)
	}
	draft = cleanOutput(draft)

	// Step 2: review — ask the model to remove anything not in the transcript.
	fmt.Fprintln(os.Stderr, "Reviewing summary...")
	reviewed, err := o.complete(
		"You are a strict fact-checker. "+
			"Read the transcript and draft summary. "+
			"Your ONLY job: remove sentences from the draft that contain information NOT in the transcript. "+
			"Do NOT rewrite, paraphrase, or add anything. "+
			"Output only the remaining sentences unchanged.",
		"Transcript:\n"+transcription+"\n\nDraft summary:\n"+draft+"\n\nCorrected (only remove invented sentences):",
	)
	if err != nil {
		// Review failed — return the draft rather than an error.
		fmt.Fprintf(os.Stderr, "Warning: review step failed: %v\n", err)
		return draft, nil
	}

	return cleanOutput(reviewed), nil
}

// complete sends a single system+user turn and returns the assistant reply.
func (o *OllamaSummarizer) complete(system, user string) (string, error) {
	req := openai.ChatCompletionRequest{
		Model:       o.model,
		MaxTokens:   defaultMaxTokens,
		Temperature: 0,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: system},
			{Role: openai.ChatMessageRoleUser, Content: user},
		},
	}
	resp, err := o.client.CreateChatCompletion(context.Background(), req)
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned")
	}
	return resp.Choices[0].Message.Content, nil
}

// cleanOutput strips CJK characters (Chinese/Japanese/Korean) that some models
// inject mid-response. When CJK is detected the model typically self-corrects and
// re-generates the answer in Russian after the CJK block — so we take the last
// non-empty paragraph from the cleaned text as the final answer.
func cleanOutput(s string) string {
	hasCJK := containsCJK(s)

	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = trimAtCJK(line)
	}
	result := strings.Join(lines, "\n")
	for strings.Contains(result, "\n\n\n") {
		result = strings.ReplaceAll(result, "\n\n\n", "\n\n")
	}
	result = strings.TrimSpace(result)

	if hasCJK {
		// Take the last non-empty paragraph — that is the model's final clean answer.
		paras := strings.Split(result, "\n\n")
		for i := len(paras) - 1; i >= 0; i-- {
			if p := strings.TrimSpace(paras[i]); p != "" {
				return p
			}
		}
	}

	return result
}

// containsCJK reports whether s contains any CJK rune.
func containsCJK(s string) bool {
	for _, r := range s {
		if isCJK(r) {
			return true
		}
	}
	return false
}

// trimAtCJK returns s truncated at the first CJK character.
// It tries to trim to the last sentence end (.!?); if none found, trims to last word boundary.
func trimAtCJK(s string) string {
	for i, r := range s {
		if isCJK(r) {
			before := s[:i]
			// Prefer cutting at last sentence-ending punctuation.
			for j := len(before) - 1; j >= 0; j-- {
				b := before[j]
				if b == '.' || b == '!' || b == '?' {
					return before[:j+1]
				}
			}
			// Fall back to last word boundary.
			if idx := strings.LastIndexByte(before, ' '); idx > 0 {
				return strings.TrimRight(before[:idx], " \t,，、")
			}
			return ""
		}
	}
	return s
}

// isCJK reports whether r is a CJK (Chinese/Japanese/Korean) rune.
func isCJK(r rune) bool {
	return (r >= 0x3000 && r <= 0x9fff) || (r >= 0xff00 && r <= 0xffef)
}
