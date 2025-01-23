package esi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/guarzo/eveapi/common"
	"github.com/guarzo/eveapi/common/model"
	"golang.org/x/oauth2"
)

// EsiService is a higher-level interface for retrieving or manipulating EVE data.
type EsiService interface {
	GetUserInfo(ctx context.Context, token *oauth2.Token) (*model.User, error)
	GetCharacterInfo(ctx context.Context, characterID int) (*model.Character, error)
	GetCharacterAssets(ctx context.Context, characterID int64, token *oauth2.Token) ([]model.LocationInventory, error)
	GetCorporationAssets(ctx context.Context, corporationID int64, token *oauth2.Token) ([]model.LocationInventory, error)
	GetCharacterLocation(ctx context.Context, characterID int64, token *oauth2.Token) (int64, error)
	GetCloneLocations(ctx context.Context, characterID int64, token *oauth2.Token) (int64, []int64, error)
	GetStructure(ctx context.Context, structureID int64, token *oauth2.Token) (*model.Structure, error)
	GetStation(ctx context.Context, stationID int64) (*model.Station, error)
	GetEsiKillMail(ctx context.Context, killID int, hash string) (*model.EsiKillMail, error)
	CharacterIDSearch(characterID int64, name string, token *oauth2.Token) (int32, error)
	CorporationIDSearch(characterID int64, name string, token *oauth2.Token) (int32, error)
	AllianceIDSearch(characterID int64, name string, token *oauth2.Token) (int32, error)
	IDSearch(characterID int64, name, category string, token *oauth2.Token) (int32, error)
	GetPublicCharacterData(characterID int64, token *oauth2.Token) (*model.CharacterResponse, error)
	GetCharacterData(characterID int64, token *oauth2.Token) (*model.CharacterResponse, error)
	GetSystemName(systemID int) string
	GetCharacterCorporation(characterID int64, token *oauth2.Token) (int32, error)
	GetCharacterPortrait(characterID int64) (string, error)
	GetCorporationInfo(ctx context.Context, corporationID int) (*model.Corporation, error)
	GetAllianceInfo(ctx context.Context, allianceID int) (*model.Alliance, error)
}

// esiService is the concrete implementation that uses an EsiClient.
type esiService struct {
	esiClient EsiClient
	cache     common.CacheRepository
	auth      AuthClient
}

// NewEsiService constructs an EsiService.
func NewEsiService(client EsiClient) EsiService {
	return &esiService{
		esiClient: client,
	}
}

// ---------------------------------------------------------------------------------------
// 1) Existing Methods
// ---------------------------------------------------------------------------------------

func (s *esiService) GetUserInfo(ctx context.Context, token *oauth2.Token) (*model.User, error) {
	if token == nil || token.AccessToken == "" {
		return nil, fmt.Errorf("no token provided")
	}

	url := "https://login.eveonline.com/oauth/verify"
	data, err := s.esiClient.DoRequest(ctx, http.MethodGet, url, token, nil)
	if err != nil {
		return nil, err
	}

	var user model.User
	if err = unmarshalJSON(data, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *esiService) GetCharacterInfo(ctx context.Context, characterID int) (*model.Character, error) {
	endpoint := fmt.Sprintf("characters/%d/", characterID)
	var char model.Character
	err := s.esiClient.GetJSON(ctx, endpoint, &char, nil, nil)
	if err != nil {
		// check if 404
		var httpErr *common.HTTPError
		if isHttpError(err, httpErr) && httpErr.StatusCode == http.StatusNotFound {
			return nil, err
		}
		return nil, err
	}
	return &char, nil
}

func (s *esiService) GetEsiKillMail(ctx context.Context, killMailID int, hash string) (*model.EsiKillMail, error) {
	endpoint := fmt.Sprintf("killmails/%d/%s/", killMailID, hash)
	var km model.EsiKillMail
	if err := s.esiClient.GetJSON(ctx, endpoint, &km, nil, nil); err != nil {
		return nil, fmt.Errorf("failed to fetch ESI killmail: %w", err)
	}
	return &km, nil
}

// ---------------------------------------------------------------------------------------
// 2) Newly Exposed Methods
// ---------------------------------------------------------------------------------------

// (A) ID search methods
func (s *esiService) CharacterIDSearch(characterID int64, name string, token *oauth2.Token) (int32, error) {
	return s.IDSearch(characterID, name, "character", token)
}

func (s *esiService) CorporationIDSearch(characterID int64, name string, token *oauth2.Token) (int32, error) {
	return s.IDSearch(characterID, name, "corporation", token)
}

func (s *esiService) AllianceIDSearch(characterID int64, name string, token *oauth2.Token) (int32, error) {
	return s.IDSearch(characterID, name, "alliance", token)
}

func (s *esiService) IDSearch(characterID int64, name, category string, token *oauth2.Token) (int32, error) {
	ctx := context.Background()
	baseURL := fmt.Sprintf("characters/%d/search/", characterID)
	params := map[string]string{
		"categories": category,
		"datasource": "tranquility",
		"language":   "en",
		"search":     name,
		"strict":     "true",
	}
	if token != nil {
		params["token"] = token.AccessToken
	}

	data, err := s.esiClient.GetBytes(ctx, baseURL, token, params)
	if err != nil {
		return 0, err
	}

	var result map[string][]int32
	if err = json.Unmarshal(data, &result); err != nil {
		return 0, fmt.Errorf("failed to parse JSON response: %v", err)
	}

	ids, exists := result[category]
	if !exists || len(ids) == 0 {
		return 0, fmt.Errorf("no IDs returned for that name")
	}

	tempID := ids[0]
	if len(ids) > 1 {
		// verify exact match
		for _, id := range ids {
			data, err := s.GetPublicCharacterData(int64(id), token)
			if err != nil {
				continue
			}
			if strings.EqualFold(data.Name, name) {
				tempID = id
				break
			}
		}
	}

	return tempID, nil
}

// (B) Character data methods
func (s *esiService) GetPublicCharacterData(characterID int64, token *oauth2.Token) (*model.CharacterResponse, error) {
	return s.GetCharacterData(characterID, token)
}

func (s *esiService) GetCharacterData(characterID int64, token *oauth2.Token) (*model.CharacterResponse, error) {
	ctx := context.Background()
	endpoint := fmt.Sprintf("characters/%d/", characterID)
	var character model.CharacterResponse
	err := s.esiClient.GetJSON(ctx, endpoint, &character, token, nil)
	if err != nil {
		return nil, err
	}
	return &character, nil
}

// (C) System name
func (s *esiService) GetSystemName(systemID int) string {
	ctx := context.Background()
	url := fmt.Sprintf("universe/systems/%d/", systemID)
	var sys struct {
		Name string `json:"name"`
	}
	_ = s.esiClient.GetJSON(ctx, url, &sys, nil, nil)
	return sys.Name
}

// (D) Misc character corp methods
func (s *esiService) GetCharacterCorporation(characterID int64, token *oauth2.Token) (int32, error) {
	data, err := s.GetCharacterData(characterID, token)
	if err != nil {
		return 0, err
	}
	return data.CorporationID, nil
}

func (s *esiService) GetCharacterPortrait(characterID int64) (string, error) {
	ctx := context.Background()
	endpoint := fmt.Sprintf("characters/%d/portrait/", characterID)

	var portrait model.CharacterPortrait
	err := s.esiClient.GetJSON(ctx, endpoint, &portrait, nil, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decode response body: %v", err)
	}

	return portrait.Px64x64, nil
}

// (E) Corporation / Alliance Info
func (s *esiService) GetCorporationInfo(ctx context.Context, corporationID int) (*model.Corporation, error) {
	var corporation model.Corporation
	endpoint := fmt.Sprintf("corporations/%d/", corporationID)
	if err := s.esiClient.GetJSON(ctx, endpoint, &corporation, nil, nil); err != nil {
		return nil, err
	}
	return &corporation, nil
}

func (s *esiService) GetAllianceInfo(ctx context.Context, allianceID int) (*model.Alliance, error) {
	if allianceID == 0 {
		return nil, fmt.Errorf("no alliance specified")
	}
	var alliance model.Alliance
	endpoint := fmt.Sprintf("alliances/%d/", allianceID)
	if err := s.esiClient.GetJSON(ctx, endpoint, &alliance, nil, nil); err != nil {
		return nil, err
	}
	return &alliance, nil
}

func isHttpError(src error, tgt *common.HTTPError) bool {
	// A simple approach that checks text in error string:
	return strings.Contains(src.Error(), "unexpected status code")
}
