package common_test

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/guarzo/eveapi/common"
)

func TestNewEveHttpClient(t *testing.T) {
	base := &http.Client{}
	client := common.NewEveHttpClient("MyUserAgent", base)
	if client == nil {
		t.Fatal("expected non-nil HttpClient")
	}
}

func TestHttpClient_Do(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "TestUserAgent" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, "wrong user-agent")
			return
		}
		fmt.Fprint(w, "hello world")
	}))
	defer ts.Close()

	base := &http.Client{}
	hc := common.NewEveHttpClient("TestUserAgent", base)

	req, err := http.NewRequest(http.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := hc.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "hello world" && resp.StatusCode == http.StatusOK {
		t.Errorf("unexpected response: %s", string(body))
	}
}

func TestHttpClient_RetryWithExponentialBackoff(t *testing.T) {
	called := 0
	operation := func() (interface{}, error) {
		called++
		if called < 3 {
			// simulate a 503
			return nil, &common.HTTPError{
				StatusCode: http.StatusServiceUnavailable,
				Body:       []byte("temporary issue"),
			}
		}
		return "success", nil
	}

	hc := common.NewEveHttpClient("UA", &http.Client{})
	// disable real sleep
	hc.SetRandAndSleepForTest(func(d time.Duration) {}, rand.Int63())

	res, err := hc.RetryWithExponentialBackoff(operation)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.(string) != "success" {
		t.Errorf("expected 'success', got %v", res)
	}
	if called != 3 {
		t.Errorf("expected 3 calls, got %d", called)
	}
}
