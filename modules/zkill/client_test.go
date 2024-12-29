package zkill_test

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/guarzo/eveapi/common"
	"github.com/guarzo/eveapi/common/model"
	"github.com/guarzo/eveapi/modules/zkill"
)

type mockEsiClient struct {
	getJSONFunc    func(ctx context.Context, endpoint string, entity interface{}, token *oauth2.Token, params map[string]string) error
	getBytesFunc   func(ctx context.Context, endpoint string, token *oauth2.Token, params map[string]string) ([]byte, error)
	doRequestFunc  func(ctx context.Context, method, urlStr string, token *oauth2.Token, body io.Reader, expectedStatus ...int) ([]byte, error)
	postJSONFunc   func(ctx context.Context, endpoint string, token *oauth2.Token, body io.Reader, expectedStatusCodes ...int) ([]byte, error)
	deleteJSONFunc func(ctx context.Context, endpoint string, token *oauth2.Token, body io.Reader, expectedStatusCodes ...int) ([]byte, error)
}

func (m *mockEsiClient) GetJSON(ctx context.Context, endpoint string, entity interface{}, token *oauth2.Token, params map[string]string) error {
	return m.getJSONFunc(ctx, endpoint, entity, token, params)
}
func (m *mockEsiClient) GetBytes(ctx context.Context, endpoint string, token *oauth2.Token, params map[string]string) ([]byte, error) {
	return m.getBytesFunc(ctx, endpoint, token, params)
}
func (m *mockEsiClient) DoRequest(ctx context.Context, method, urlStr string, token *oauth2.Token, body io.Reader, expectedStatus ...int) ([]byte, error) {
	return m.doRequestFunc(ctx, method, urlStr, token, body, expectedStatus...)
}
func (m *mockEsiClient) PostJSON(ctx context.Context, endpoint string, token *oauth2.Token, body io.Reader, expectedStatusCodes ...int) ([]byte, error) {
	return m.postJSONFunc(ctx, endpoint, token, body, expectedStatusCodes...)
}
func (m *mockEsiClient) DeleteJSON(ctx context.Context, endpoint string, token *oauth2.Token, body io.Reader, expectedStatusCodes ...int) ([]byte, error) {
	return m.deleteJSONFunc(ctx, endpoint, token, body, expectedStatusCodes...)
}

type mockCache struct {
	store map[string][]byte
}

func (m *mockCache) Get(key string) ([]byte, bool) {
	val, ok := m.store[key]
	return val, ok
}
func (m *mockCache) Set(key string, value []byte, _ time.Duration) {
	m.store[key] = value
}
func (m *mockCache) Delete(key string) {
	delete(m.store, key)
}

type mockLogger struct{}

func (l *mockLogger) Debugf(format string, args ...interface{}) {}
func (l *mockLogger) Infof(format string, args ...interface{})  {}
func (l *mockLogger) Warnf(format string, args ...interface{})  {}
func (l *mockLogger) Errorf(format string, args ...interface{}) {}

func TestZKillClient_GetKillsPageData_Cached(t *testing.T) {
	testMails := []model.ZkillMail{
		{KillMailID: 123, ZKB: model.ZKB{Hash: "abc", TotalValue: 1000}},
	}
	data, _ := json.Marshal(testMails)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, string(data))
	}))
	defer ts.Close()

	c := &mockCache{store: make(map[string][]byte)}
	cli := zkill.NewZkillClient(ts.URL, common.NewEveHttpClient("UA", &http.Client{}), c)

	ctx := context.Background()
	result, err := cli.GetKillsPageData(ctx, "character", 999, 1, 2023, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1, got %d", len(result))
	}
	// second call => from cache
	res2, _ := cli.GetKillsPageData(ctx, "character", 999, 1, 2023, 10)
	if len(res2) != 1 {
		t.Errorf("expected 1 from cache, got %d", len(res2))
	}
}
