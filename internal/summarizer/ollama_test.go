package summarizer_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/falkomer/meet-summarize/internal/summarizer"
)

// TestNewOllamaSummarizer_Defaults verifies that NewOllamaSummarizer with empty baseURL
// and model still returns a non-nil summarizer that satisfies the Summarizer interface.
func TestNewOllamaSummarizer_Defaults(t *testing.T) {
	s := summarizer.NewOllamaSummarizer("test-key", "", "")
	if s == nil {
		t.Fatal("expected non-nil OllamaSummarizer, got nil")
	}
	// Ensure it implements the Summarizer interface at compile time.
	var _ summarizer.Summarizer = s
}

// TestNewOllamaSummarizer_CustomValues verifies that NewOllamaSummarizer with explicit
// baseURL and model returns a non-nil summarizer.
func TestNewOllamaSummarizer_CustomValues(t *testing.T) {
	s := summarizer.NewOllamaSummarizer("test-key", "https://custom.example.com/v1/", "custom-model")
	if s == nil {
		t.Fatal("expected non-nil OllamaSummarizer, got nil")
	}
	var _ summarizer.Summarizer = s
}

// TestOllamaSummarizer_Summarize_MockServer verifies that Summarize correctly parses
// an OpenAI-compatible response and returns the assistant message content.
func TestOllamaSummarizer_Summarize_MockServer(t *testing.T) {
	mockResponse := `{
		"id": "chatcmpl-123",
		"object": "chat.completion",
		"choices": [{
			"index": 0,
			"message": {"role": "assistant", "content": "Meeting summary: discussed project timeline."},
			"finish_reason": "stop"
		}],
		"usage": {"prompt_tokens": 10, "completion_tokens": 20, "total_tokens": 30}
	}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, mockResponse)
	}))
	defer srv.Close()

	baseURL := srv.URL + "/"
	s := summarizer.NewOllamaSummarizer("test-key", baseURL, "test-model")

	result, err := s.Summarize("Alice: hello\nBob: hi", "Summarize the meeting.")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	expected := "Meeting summary: discussed project timeline."
	if result != expected {
		t.Errorf("expected result %q, got %q", expected, result)
	}
}

// TestOllamaSummarizer_Summarize_APIError verifies that a non-2xx response from the
// API causes Summarize to return a non-nil error.
func TestOllamaSummarizer_Summarize_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error": {"message": "internal server error", "type": "server_error"}}`)
	}))
	defer srv.Close()

	baseURL := srv.URL + "/"
	s := summarizer.NewOllamaSummarizer("test-key", baseURL, "test-model")

	_, err := s.Summarize("some transcript", "some prompt")
	if err == nil {
		t.Fatal("expected an error from API 500 response, got nil")
	}
	if !strings.Contains(err.Error(), "Ollama summarization request failed") {
		t.Errorf("expected error to contain 'Ollama summarization request failed', got: %v", err)
	}
}

// TestOllamaSummarizer_Summarize_EmptyChoices verifies that a 200 response with an
// empty choices array causes Summarize to return a non-nil error.
func TestOllamaSummarizer_Summarize_EmptyChoices(t *testing.T) {
	mockResponse := `{
		"id": "chatcmpl-456",
		"object": "chat.completion",
		"choices": [],
		"usage": {"prompt_tokens": 5, "completion_tokens": 0, "total_tokens": 5}
	}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, mockResponse)
	}))
	defer srv.Close()

	baseURL := srv.URL + "/"
	s := summarizer.NewOllamaSummarizer("test-key", baseURL, "test-model")

	_, err := s.Summarize("transcript", "prompt")
	if err == nil {
		t.Fatal("expected an error for empty choices, got nil")
	}
	if !strings.Contains(err.Error(), "no choices") {
		t.Errorf("expected error to contain 'no choices', got: %v", err)
	}
}
