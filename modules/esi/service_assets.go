package esi

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"

	"github.com/guarzo/eveapi/common/model"
)

// This file focuses on asset endpoints and cyno logic.

// CynoItems might be defined globally here or in your service.
var CynoItems = []model.Item{
	{ID: 16273, Name: "Liquid Ozone", Qty: 200},
	{ID: 32880, Name: "Venture", Qty: 1},
	{ID: 19744, Name: "Covetor", Qty: 1},
}

// GetCharacterAssets calls ESI’s /characters/{id}/assets/
func (s *esiService) GetCharacterAssets(ctx context.Context, characterID int64, token *oauth2.Token) ([]model.LocationInventory, error) {
	rawAssets, err := s.fetchAssets(ctx, fmt.Sprintf("characters/%d", characterID), token)
	if err != nil {
		return nil, err
	}
	locItems := groupAssetsByLocation(rawAssets)
	cynoMap := createCynoMap(CynoItems)

	var results []model.LocationInventory
	for locID, assets := range locItems {
		itemsInLoc := summarizeItemsInLocation(assets)
		if hasRequiredCynoItems(itemsInLoc, cynoMap) {
			inv := buildLocationInventory(characterID, int64(locID), assets)
			results = append(results, inv)
		}
	}
	return results, nil
}

// GetCorporationAssets calls ESI’s /corporations/{id}/assets/
func (s *esiService) GetCorporationAssets(ctx context.Context, corpID int64, token *oauth2.Token) ([]model.LocationInventory, error) {
	rawAssets, err := s.fetchAssets(ctx, fmt.Sprintf("corporations/%d", corpID), token)
	if err != nil {
		return nil, err
	}
	locItems := groupAssetsByLocation(rawAssets)
	cynoMap := createCynoMap(CynoItems)

	var results []model.LocationInventory
	for locID, assets := range locItems {
		itemsInLoc := summarizeItemsInLocation(assets)
		if hasRequiredCynoItems(itemsInLoc, cynoMap) {
			inv := buildLocationInventory(corpID, int64(locID), assets)
			results = append(results, inv)
		}
	}
	return results, nil
}

// fetchAssets uses EsiClient.GetJSON to get an array of model.Asset
func (s *esiService) fetchAssets(ctx context.Context, path string, token *oauth2.Token) ([]model.Asset, error) {
	endpoint := fmt.Sprintf("%s/assets/?datasource=tranquility", path)
	var out []model.Asset
	err := s.esiClient.GetJSON(ctx, endpoint, &out, token, nil)
	return out, err
}

// group them by location
func groupAssetsByLocation(raw []model.Asset) map[int][]model.Asset {
	m := make(map[int][]model.Asset)
	for _, asset := range raw {
		if isRelevantLocation(asset.LocationType) {
			lid := int(asset.LocationID)
			m[lid] = append(m[lid], asset)
		}
	}
	return m
}

func isRelevantLocation(locType string) bool {
	return locType == "station" || locType == "solar_system" || locType == "structure"
}

// summarize item counts
func summarizeItemsInLocation(assets []model.Asset) map[int64]int {
	counts := make(map[int64]int)
	for _, a := range assets {
		counts[a.TypeID] += a.Quantity
	}
	return counts
}

// Check if we have at least 1 cyno item
func hasRequiredCynoItems(items map[int64]int, cyno map[int64]int) bool {
	for itemID, needed := range cyno {
		if items[itemID] >= needed {
			return true
		}
	}
	return false
}

func buildLocationInventory(ownerID, locID int64, assets []model.Asset) model.LocationInventory {
	invMap := make(map[string]int)
	var locFlag, locType string

	for _, a := range assets {
		if cynoName, ok := getCynoItemName(a.TypeID); ok {
			invMap[cynoName] += a.Quantity
			locFlag = a.LocationFlag
			locType = a.LocationType
		}
	}

	return model.LocationInventory{
		CharacterID: ownerID, // if it’s corp, we can rename. But we’ll keep the field name for now.
		LocFlag:     locFlag,
		LocType:     locType,
		LocID:       int(locID),
		Items:       invMap,
	}
}

func getCynoItemName(typeID int64) (string, bool) {
	for _, it := range CynoItems {
		if it.ID == typeID {
			return it.Name, true
		}
	}
	return "", false
}

// create map of cyno items from a []model.Item
func createCynoMap(items []model.Item) map[int64]int {
	m := make(map[int64]int)
	for _, i := range items {
		m[i.ID] = i.Qty
	}
	return m
}
