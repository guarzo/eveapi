package esi

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/guarzo/eveapi/common"
	"github.com/guarzo/eveapi/common/model"
	"golang.org/x/oauth2"
)

// EsiService is a higher-level interface for retrieving or manipulating EVE data.
type EsiService interface {
	// Example: from the older code
	GetUserInfo(ctx context.Context, token *oauth2.Token) (*model.User, error)
	GetCharacterInfo(ctx context.Context, characterID int) (*model.Character, error)
	// New asset endpoints
	GetCharacterAssets(ctx context.Context, characterID int64, token *oauth2.Token) ([]model.LocationInventory, error)
	GetCorporationAssets(ctx context.Context, corporationID int64, token *oauth2.Token) ([]model.LocationInventory, error)
	// New location endpoints
	GetCharacterLocation(ctx context.Context, characterID int64, token *oauth2.Token) (int64, error)
	GetCloneLocations(ctx context.Context, characterID int64, token *oauth2.Token) (int64, []int64, error)
	GetStructure(ctx context.Context, structureID int64, token *oauth2.Token) (*model.Structure, error)
	GetStation(ctx context.Context, stationID int64) (*model.Station, error)
	// etc...
}

// esiService is the concrete implementation that uses EsiClient.
type esiService struct {
	logger    common.Logger
	esiClient EsiClient

	// if you have to store something like "failed characters" or local caches:
	mu sync.Mutex
}

// NewEsiService constructs an EsiService.
func NewEsiService(logger common.Logger, client EsiClient) EsiService {
	return &esiService{
		logger:    logger,
		esiClient: client,
	}
}

// Example: from older code: /oauth/verify
func (s *esiService) GetUserInfo(ctx context.Context, token *oauth2.Token) (*model.User, error) {
	if token == nil || token.AccessToken == "" {
		return nil, fmt.Errorf("no token provided")
	}

	// This endpoint is outside normal ESI pattern, so we do a direct request
	url := "https://login.eveonline.com/oauth/verify"
	data, err := s.esiClient.DoRequest(ctx, http.MethodGet, url, token, nil)
	if err != nil {
		return nil, err
	}

	var user model.User
	if err := unmarshalJSON(data, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// Example: minimal getCharacterInfo
func (s *esiService) GetCharacterInfo(ctx context.Context, characterID int) (*model.Character, error) {
	endpoint := fmt.Sprintf("characters/%d/", characterID)
	var char model.Character
	err := s.esiClient.GetJSON(ctx, endpoint, &char, nil, nil)
	if err != nil {
		// check if 404
		var httpErr *common.HTTPError
		if isHttpError(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
			s.logger.Warnf("Character %d not found", characterID)
			return nil, err
		}
		return nil, err
	}
	return &char, nil
}

// isHttpError helps with type-asserting your custom HTTPError
func isHttpError(src error, tgt **common.HTTPError) bool {
	return strings.Contains(src.Error(), "unexpected status code") // or use errors.As
}
