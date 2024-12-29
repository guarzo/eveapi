package common

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"time"
)

// HttpClient is an interface for HTTP operations with optional retry logic.
// This allows mocking or custom transport layers in testing.
type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
	Get(url string) (*http.Response, error)
	Post(url, contentType string, body io.Reader) (*http.Response, error)
	PostForm(url string, data url.Values) (*http.Response, error)
	Head(url string) (*http.Response, error)
	CloseIdleConnections()
	RetryWithExponentialBackoff(operation func() (interface{}, error)) (interface{}, error)
	SetRandAndSleepForTest(sleep func(d time.Duration), seed int64)
}

// HTTPError is a custom error that captures unexpected status codes and response bodies.
type HTTPError struct {
	StatusCode int
	Body       []byte
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("unexpected status code: %d, body: %s", e.StatusCode, string(e.Body))
}

// userAgentRoundTripper is a custom RoundTripper that adds a User-Agent header.
type userAgentRoundTripper struct {
	Wrapped   http.RoundTripper
	UserAgent string
}

func (rt *userAgentRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// clone request to avoid mutating the original
	clone := req.Clone(req.Context())
	clone.Header.Set("User-Agent", rt.UserAgent)
	return rt.Wrapped.RoundTrip(clone)
}

// Implementation of HttpClient that wraps a standard *http.Client with retry logic.
type httpClient struct {
	client    *http.Client
	sleepFunc func(d time.Duration)
}

// NewEveHttpClient returns a new HttpClient with a default 10s timeout, plus a custom User-Agent.
func NewEveHttpClient(userAgent string, base *http.Client) HttpClient {
	if base.Transport == nil {
		base.Transport = http.DefaultTransport
	}
	base.Transport = &userAgentRoundTripper{
		Wrapped:   base.Transport,
		UserAgent: userAgent,
	}
	base.Timeout = 10 * time.Second

	return &httpClient{
		client:    base,
		sleepFunc: time.Sleep,
	}
}

// Implementation of the interface:

func (h *httpClient) Do(req *http.Request) (*http.Response, error) {
	return h.client.Do(req)
}

func (h *httpClient) Get(url string) (*http.Response, error) {
	return h.client.Get(url)
}

func (h *httpClient) Post(url, contentType string, body io.Reader) (*http.Response, error) {
	return h.client.Post(url, contentType, body)
}

func (h *httpClient) PostForm(url string, data url.Values) (*http.Response, error) {
	return h.client.PostForm(url, data)
}

func (h *httpClient) Head(url string) (*http.Response, error) {
	return h.client.Head(url)
}

func (h *httpClient) CloseIdleConnections() {
	h.client.CloseIdleConnections()
}

// Exponential backoff constants
const (
	maxRetries = 5
	baseDelay  = 1 * time.Second
	maxDelay   = 32 * time.Second
)

// RetryWithExponentialBackoff attempts the given operation() multiple times if
// we encounter a retryable HTTPError (5xx, etc.). Adjust logic to match your needs.
func (h *httpClient) RetryWithExponentialBackoff(operation func() (interface{}, error)) (interface{}, error) {
	var result interface{}
	var err error
	delay := baseDelay

	for i := 0; i < maxRetries; i++ {
		if result, err = operation(); err == nil {
			return result, nil
		}

		var httpErr *HTTPError
		if errors.As(err, &httpErr) {
			// Check status for retry
			if httpErr.StatusCode == http.StatusInternalServerError ||
				httpErr.StatusCode == http.StatusBadGateway ||
				httpErr.StatusCode == http.StatusServiceUnavailable ||
				httpErr.StatusCode == http.StatusGatewayTimeout {

				if i == maxRetries-1 {
					break
				}
				// apply jitter
				jitter := time.Duration(rand.Int63n(int64(delay)))
				h.sleepFunc(delay + jitter)

				delay *= 2
				if delay > maxDelay {
					delay = maxDelay
				}
				continue
			}
		}
		// Not retryable, break
		break
	}
	return nil, err
}

func (h *httpClient) SetRandAndSleepForTest(sleep func(d time.Duration), seed int64) {
	h.sleepFunc = sleep
	rand.Seed(seed)
}
