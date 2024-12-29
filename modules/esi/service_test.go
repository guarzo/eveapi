package esi_test

import (
	"context"
	"errors"
	"golang.org/x/oauth2"
	"io"
	"reflect"
	"testing"

	"github.com/guarzo/eveapi/common/model"
	"github.com/guarzo/eveapi/modules/esi"
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

func TestEsiService_GetUserInfo(t *testing.T) {
	mClient := &mockEsiClient{
		doRequestFunc: func(ctx context.Context, method, urlStr string, token *oauth2.Token, body io.Reader, expectedStatus ...int) ([]byte, error) {
			if urlStr != "https://login.eveonline.com/oauth/verify" {
				return nil, errors.New("unexpected URL in doRequest")
			}
			return []byte(`{"CharacterID":123,"CharacterName":"Test Char"}`), nil
		},
	}

	svc := esi.NewEsiService(mClient)

	ctx := context.Background()
	user, err := svc.GetUserInfo(ctx, &oauth2.Token{AccessToken: "abc"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := &model.User{CharacterID: 123, CharacterName: "Test Char"}
	if !reflect.DeepEqual(user, expected) {
		t.Errorf("got %#v, want %#v", user, expected)
	}
}
