package moderation

import "context"

type TextResult struct {
	Status     string
	Categories []string
	Scores     map[string]float64
}

type ImageResult struct {
	Status     string
	Confidence float64
}

type TextChecker interface {
	Check(ctx context.Context, text string) (TextResult, error)
}

type ImageChecker interface {
	Check(ctx context.Context, fileName string, fileBytes []byte) (ImageResult, error)
}
