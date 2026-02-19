package summarizer

type Summarizer interface {
	Summarize(transcription string, prompt string) (string, error)
}
