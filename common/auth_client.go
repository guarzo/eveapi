package common

import "golang.org/x/oauth2"

// AuthClient defines the ability to refresh an OAuth2 token.
// If you have more complex auth flows (e.g. client credentials, password grant),
// you might add more methods here.
type AuthClient interface {
	// RefreshToken attempts to refresh using the given refresh token string.
	// Returns a new *oauth2.Token on success, or an error if refresh fails.
	RefreshToken(refreshToken string) (*oauth2.Token, error)
}
