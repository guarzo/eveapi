package esi

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"golang.org/x/oauth2"

	"github.com/guarzo/eveapi/common/model"
)

// We’ll keep a small local map-based cache for station/structure lookups
var (
	locCache  = make(map[int64]int64)
	locCacheM sync.RWMutex
)

// GetCharacterLocation calls ESI /characters/{id}/location/
func (s *esiService) GetCharacterLocation(ctx context.Context, characterID int64, token *oauth2.Token) (int64, error) {
	endpoint := fmt.Sprintf("characters/%d/location/?datasource=tranquility", characterID)
	var loc model.CharacterLocation
	err := s.esiClient.GetJSON(ctx, endpoint, &loc, token, nil)
	if err != nil {
		return 0, err
	}
	return loc.SolarSystemID, nil
}

// GetCloneLocations calls ESI /characters/{id}/clones/
func (s *esiService) GetCloneLocations(ctx context.Context, characterID int64, token *oauth2.Token) (int64, []int64, error) {
	endpoint := fmt.Sprintf("characters/%d/clones/?datasource=tranquility", characterID)
	var cl model.CloneLocation
	if err := s.esiClient.GetJSON(ctx, endpoint, &cl, token, nil); err != nil {
		return 0, nil, err
	}

	homeSystem, err := s.resolveLocationSystemID(ctx, cl.HomeLocation.LocationID, cl.HomeLocation.LocationType, token)
	if err != nil {
		return 0, nil, err
	}

	var out []int64
	out = append(out, homeSystem)
	for _, jc := range cl.JumpClones {
		sysID, err := s.resolveLocationSystemID(ctx, jc.LocationID, jc.LocationType, token)
		if err != nil {
			return 0, nil, err
		}
		out = append(out, sysID)
	}
	return homeSystem, out, nil
}

// resolveLocationSystemID determines the system an ID belongs to (station or structure).
func (s *esiService) resolveLocationSystemID(ctx context.Context, locationID int64, locType string, token *oauth2.Token) (int64, error) {
	// check local cache
	if sysID, ok := s.getCache(locationID); ok {
		return sysID, nil
	}

	if locType == "structure" {
		strct, err := s.GetStructure(ctx, locationID, token)
		if err != nil {
			return 0, err
		}
		s.setCache(locationID, strct.SystemID)
		return strct.SystemID, nil
	}

	// default to "station"
	stn, err := s.GetStation(ctx, locationID)
	if err != nil {
		return 0, err
	}
	s.setCache(locationID, stn.SystemID)
	return stn.SystemID, nil
}

// GetStructure uses ESI /universe/structures/{structure_id}
func (s *esiService) GetStructure(ctx context.Context, structureID int64, token *oauth2.Token) (*model.Structure, error) {
	// check local cache
	if cached, ok := s.getCache(structureID); ok {
		return &model.Structure{SystemID: cached}, nil
	}

	endpoint := fmt.Sprintf("universe/structures/%d/?datasource=tranquility", structureID)
	var strct model.Structure
	err := s.esiClient.GetJSON(ctx, endpoint, &strct, token, nil)
	if err != nil {
		return nil, err
	}
	s.setCache(structureID, strct.SystemID)
	return &strct, nil
}

// GetStation uses ESI /universe/stations/{station_id}
func (s *esiService) GetStation(ctx context.Context, stationID int64) (*model.Station, error) {
	if cached, ok := s.getCache(stationID); ok {
		return &model.Station{SystemID: cached, ID: stationID}, nil
	}

	endpoint := fmt.Sprintf("universe/stations/%d/?datasource=tranquility", stationID)
	// We can do a direct GET if it’s public data
	data, err := s.esiClient.GetBytes(ctx, endpoint, nil, nil)
	if err != nil {
		return nil, err
	}
	var stn model.Station
	if err := json.Unmarshal(data, &stn); err != nil {
		return nil, err
	}
	s.setCache(stationID, stn.SystemID)
	return &stn, nil
}

// local cache get/set
func (s *esiService) getCache(key int64) (int64, bool) {
	locCacheM.RLock()
	defer locCacheM.RUnlock()
	val, ok := locCache[key]
	return val, ok
}

func (s *esiService) setCache(key, val int64) {
	locCacheM.Lock()
	defer locCacheM.Unlock()
	locCache[key] = val
}
