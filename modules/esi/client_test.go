package esi_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"github.com/guarzo/eveapi/modules/esi"
)

type mockHttpClient struct {
	doFunc    func(req *http.Request) (*http.Response, error)
	retryFunc func(operation func() (interface{}, error)) (interface{}, error)
	sleepFunc func(d time.Duration)
}

func (m *mockHttpClient) Do(req *http.Request) (*http.Response, error) {
	return m.doFunc(req)
}
func (m *mockHttpClient) Get(url string) (*http.Response, error) {
	panic("Get not implemented in mock")
}
func (m *mockHttpClient) Post(url, contentType string, body io.Reader) (*http.Response, error) {
	panic("Post not implemented in mock")
}
func (m *mockHttpClient) PostForm(u string, data url.Values) (*http.Response, error) {
	panic("PostForm not implemented in mock")
}
func (m *mockHttpClient) Head(url string) (*http.Response, error) {
	panic("Head not implemented in mock")
}
func (m *mockHttpClient) CloseIdleConnections() {}
func (m *mockHttpClient) RetryWithExponentialBackoff(op func() (interface{}, error)) (interface{}, error) {
	if m.retryFunc != nil {
		return m.retryFunc(op)
	}
	// default: call op directly
	return op()
}
func (m *mockHttpClient) SetRandAndSleepForTest(sleep func(d time.Duration), seed int64) {
	m.sleepFunc = sleep
}

type mockCache struct {
	store map[string][]byte
}

func (c *mockCache) Get(key string) ([]byte, bool) {
	val, ok := c.store[key]
	return val, ok
}
func (c *mockCache) Set(key string, value []byte, _ time.Duration) {
	c.store[key] = value
}
func (c *mockCache) Delete(key string) {
	delete(c.store, key)
}

type mockAuth struct {
	refreshFunc func(refreshToken string) (*oauth2.Token, error)
}

func (m *mockAuth) RefreshToken(refreshToken string) (*oauth2.Token, error) {
	if m.refreshFunc != nil {
		return m.refreshFunc(refreshToken)
	}
	return nil, errors.New("mockAuth called refresh, but no func set")
}

func TestEsiClient_DoRequest_Success(t *testing.T) {
	mockHTTP := &mockHttpClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			// Return a 200 with dummy JSON
			body := io.NopCloser(bytes.NewBufferString(`{"foo":"bar"}`))
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       body,
			}, nil
		},
	}

	mockCacheRepo := &mockCache{store: make(map[string][]byte)}
	mockAuthClient := &mockAuth{
		refreshFunc: func(token string) (*oauth2.Token, error) {
			return nil, errors.New("should not refresh token for 200 response")
		},
	}

	client := esi.NewEsiClient(
		"https://esi.evetech.net/latest/",
		mockHTTP,
		mockCacheRepo,
		mockAuthClient,
	)

	ctx := context.Background()
	data, err := client.DoRequest(ctx, http.MethodGet, "https://example.com/test", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != `{"foo":"bar"}` {
		t.Errorf("expected %v, got %v", `{"foo":"bar"}`, string(data))
	}
}

func TestEsiClient_DoRequest_Refresh(t *testing.T) {
	firstCall := true
	mockHTTP := &mockHttpClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			if firstCall {
				firstCall = false
				// simulate 403
				return &http.Response{
					StatusCode: http.StatusForbidden,
					Body:       io.NopCloser(bytes.NewBufferString("forbidden")),
				}, nil
			}
			// second call is 200
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{"refreshed":"token"}`)),
			}, nil
		},
	}

	mockCacheRepo := &mockCache{store: make(map[string][]byte)}
	mockAuthClient := &mockAuth{
		refreshFunc: func(r string) (*oauth2.Token, error) {
			// simulate success
			return &oauth2.Token{
				AccessToken:  "newAccessToken",
				RefreshToken: "newRefreshToken",
			}, nil
		},
	}

	client := esi.NewEsiClient(
		"https://esi.evetech.net/latest/",
		mockHTTP,
		mockCacheRepo,
		mockAuthClient,
	)

	ctx := context.Background()
	token := &oauth2.Token{
		AccessToken:  "oldAccessToken",
		RefreshToken: "oldRefreshToken",
	}
	data, err := client.DoRequest(ctx, http.MethodGet, "https://example.com/test", token, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != `{"refreshed":"token"}` {
		t.Errorf("expected %v, got %v", `{"refreshed":"token"}`, string(data))
	}
}

func TestEsiClient_GetBytes_Caching(t *testing.T) {
	called := 0
	mockHTTP := &mockHttpClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			called++
			body := io.NopCloser(bytes.NewBufferString(`{"cached":"data"}`))
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       body,
			}, nil
		},
	}
	mockCacheRepo := &mockCache{store: make(map[string][]byte)}
	mockAuthClient := &mockAuth{}

	client := esi.NewEsiClient("https://esi.evetech.net/latest/", mockHTTP, mockCacheRepo, mockAuthClient)

	ctx := context.Background()
	// first call
	_, err := client.GetBytes(ctx, "test/endpoint", nil, map[string]string{"datasource": "tranquility"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called != 1 {
		t.Errorf("expected called=1, got %d", called)
	}

	// second call => should use cache
	_, err = client.GetBytes(ctx, "test/endpoint", nil, map[string]string{"datasource": "tranquility"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called != 1 {
		t.Errorf("expected called=1 after second call, got %d", called)
	}
}
