package mlclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	answerPath           = "/api/v1/chat"
	maxResponseBodyBytes = 1 << 20 // 1 MiB
)

type Client interface {
	Answer(ctx context.Context, request AnswerRequest) (*AnswerResponse, error)
}
type HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

func (c *HTTPClient) Answer(ctx context.Context, request AnswerRequest) (*AnswerResponse, error) {
	payload, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal ML request: %w", err)
	}
	httpRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+answerPath,
		bytes.NewReader(payload),
	)
	if err != nil {
		return nil, fmt.Errorf("create ML request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Accept", "application/json")

	httpResponse, err := c.httpClient.Do(httpRequest)
	if err != nil {
		return nil, classifyRequestError(ctx, err)
	}
	defer httpResponse.Body.Close()

	body, err := io.ReadAll(
		io.LimitReader(httpResponse.Body, maxResponseBodyBytes+1),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: read response body: %v", ErrUnavailable, err)
	}
	if len(body) > maxResponseBodyBytes {
		return nil, fmt.Errorf(
			"%w: response body exceeds %d bytes",
			ErrInvalidResponse,
			maxResponseBodyBytes,
		)
	}
	if httpResponse.StatusCode < http.StatusOK ||
		httpResponse.StatusCode >= http.StatusMultipleChoices {
		kind := ErrUnavailable
		switch httpResponse.StatusCode {
		case 422:
			kind = ErrRejected
		case 502:
			kind = ErrInvalidResponse
		case 504:
			kind = ErrTimeout
		case 503:
			var detail struct{ Detail struct{ Code string } }
			if json.Unmarshal(body, &detail) == nil && detail.Detail.Code == "ml_busy" {
				kind = ErrBusy
			}
		}
		return nil, fmt.Errorf(
			"%w: ML service returned status %d",
			kind,
			httpResponse.StatusCode,
		)
	}
	var response AnswerResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("%w: decode JSON: %v", ErrInvalidResponse, err)
	}
	response.Answer = strings.TrimSpace(response.Answer)
	if utf8.RuneCountInString(response.Answer) > 10000 {
		return nil, fmt.Errorf("%w: answer exceeds history limit", ErrInvalidResponse)
	}
	if response.Answer == "" {
		return nil, fmt.Errorf("%w: empty answer", ErrInvalidResponse)
	}
	cleanSources := make([]string, 0, len(response.Sources))
	for _, source := range response.Sources {
		source = strings.TrimSpace(source)
		if source != "" {
			cleanSources = append(cleanSources, source)
		}
	}
	response.Sources = cleanSources
	if response.ResponseTimeMS < 0 {
		return nil, fmt.Errorf("%w: negative response time", ErrInvalidResponse)
	}
	return &response, nil
}

func New(baseURL string, timeout time.Duration) (*HTTPClient, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, errors.New("ML_SERVICE_URL is empty")
	}

	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse ML_SERVICE_URL: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, errors.New("ML_SERVICE_URL must use http or https")
	}
	if parsedURL.Host == "" {
		return nil, errors.New("ML_SERVICE_URL must contain host")
	}
	if timeout <= 0 {
		return nil, errors.New("ML request timeout must be positive")
	}

	return &HTTPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

func classifyRequestError(ctx context.Context, err error) error {
	if errors.Is(ctx.Err(), context.Canceled) {
		return context.Canceled
	}

	if errors.Is(ctx.Err(), context.DeadlineExceeded) ||
		errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("%w: %v", ErrTimeout, err)
	}

	var networkError interface {
		Timeout() bool
	}
	if errors.As(err, &networkError) && networkError.Timeout() {
		return fmt.Errorf("%w: %v", ErrTimeout, err)
	}

	return fmt.Errorf("%w: %v", ErrUnavailable, err)
}
