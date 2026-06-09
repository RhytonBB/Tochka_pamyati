package moderation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

type HTTPTextChecker struct {
	baseURL string
	client  *http.Client
}

func NewHTTPTextChecker(baseURL string) *HTTPTextChecker {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	return &HTTPTextChecker{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 3 * time.Second},
	}
}

func (c *HTTPTextChecker) Check(ctx context.Context, text string) (TextResult, error) {
	if c.baseURL == "" {
		return TextResult{}, errors.New("text checker disabled")
	}

	body, _ := json.Marshal(map[string]string{"text": text})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/filter", bytes.NewReader(body))
	if err != nil {
		return TextResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return TextResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return TextResult{}, errors.New(resp.Status)
	}

	var out struct {
		Status     string             `json:"status"`
		Categories []string           `json:"categories"`
		Scores     map[string]float64 `json:"scores"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return TextResult{}, err
	}
	return TextResult{Status: out.Status, Categories: out.Categories, Scores: out.Scores}, nil
}

type HTTPImageChecker struct {
	baseURL string
	client  *http.Client
}

func NewHTTPImageChecker(baseURL string) *HTTPImageChecker {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	return &HTTPImageChecker{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *HTTPImageChecker) Check(ctx context.Context, fileName string, fileBytes []byte) (ImageResult, error) {
	if c.baseURL == "" {
		return ImageResult{}, errors.New("image checker disabled")
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("file", fileName)
	if err != nil {
		_ = w.Close()
		return ImageResult{}, err
	}
	if _, err := io.Copy(part, bytes.NewReader(fileBytes)); err != nil {
		_ = w.Close()
		return ImageResult{}, err
	}
	if err := w.Close(); err != nil {
		return ImageResult{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/filter", &buf)
	if err != nil {
		return ImageResult{}, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := c.client.Do(req)
	if err != nil {
		return ImageResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ImageResult{}, errors.New(resp.Status)
	}

	var out struct {
		Status     string  `json:"status"`
		Confidence float64 `json:"confidence"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return ImageResult{}, err
	}
	return ImageResult{Status: out.Status, Confidence: out.Confidence}, nil
}
