package model

import (
	"encoding/json"
	"golang.org/x/oauth2"
)

// For JSON handling if you want a central place to do it
func UnmarshalJSON(data []byte, out interface{}) error {
	return json.Unmarshal(data, out)
}

// Add as many shared EVE data structures as needed here.

// Example from your older code:

type Character struct {
	User
	Location       int64   `json:"Location"`
	HomeLocation   int64   `json:"HomeLocation"`
	CorporationID  int64   `json:"CorporationID"`
	CloneLocations []int64 `json:"CloneLocations"`
	CharacterRoles
	StashList []Stash `json:"StashList"`
}

type CharacterData struct {
	Token oauth2.Token
	Character
}

type User struct {
	CharacterID   int64  `json:"CharacterID"`
	CharacterName string `json:"CharacterName"`
}

// E.g. Roles
type CharacterRoles struct {
	Roles        []string `json:"roles"`
	RolesAtBase  []string `json:"roles_at_base"`
	RolesAtHQ    []string `json:"roles_at_hq"`
	RolesAtOther []string `json:"roles_at_other"`
}

// E.g. Assets
type Asset struct {
	TypeID       int64  `json:"type_id"`
	Quantity     int    `json:"quantity"`
	LocationFlag string `json:"location_flag"`
	LocationType string `json:"location_type"`
	LocationID   int64  `json:"location_id"`
}

type Item struct {
	ID   int64  `json:"item_id"`
	Name string `json:"item_name"`
	Qty  int    `json:"item_qty"`
}

type LocationInventory struct {
	CharacterID int64          `json:"Id"`
	LocFlag     string         `json:"LocFlag"`
	LocType     string         `json:"LocType"`
	LocID       int            `json:"LocID"`
	Items       map[string]int `json:"Items"`
}

type Stash struct {
	SystemId   int64  `json:"system_id"`
	SystemName string `json:"system_name"`
	Inventory  []Item `json:"inventory"`
}

type Station struct {
	SystemID int64  `json:"system_id"`
	ID       int64  `json:"station_id"`
	Name     string `json:"station_name"`
}

type Structure struct {
	Name     string `json:"name"`
	OwnerID  int64  `json:"owner_id"`
	SystemID int64  `json:"solar_system_id"`
	TypeID   int64  `json:"type_id"`
}

type CharacterLocation struct {
	SolarSystemID int64 `json:"solar_system_id"`
	StructureID   int64 `json:"structure_id"`
}

type CloneLocation struct {
	HomeLocation struct {
		LocationID   int64  `json:"location_id"`
		LocationType string `json:"location_type"`
	} `json:"home_location"`
	JumpClones []struct {
		Implants     []int  `json:"implants"`
		JumpCloneID  int64  `json:"jump_clone_id"`
		LocationID   int64  `json:"location_id"`
		LocationType string `json:"location_type"`
	} `json:"jump_clones"`
}

type ZkillMail struct {
	KillMailID int64 `json:"killmail_id"`
	ZKB        ZKB   `json:"zkb"`
}

type ZKB struct {
	Hash           string  `json:"hash"`
	FittedValue    float64 `json:"fittedValue"`
	DroppedValue   float64 `json:"droppedValue"`
	DestroyedValue float64 `json:"destroyedValue"`
	TotalValue     float64 `json:"totalValue"`
	Points         int     `json:"points"`
	NPC            bool    `json:"npc"`
	Solo           bool    `json:"solo"`
	Awox           bool    `json:"awox"`
}

// FlattenedKillMail merges Zkill data + ESI data
type FlattenedKillMail struct {
	KillMailID     int64
	Hash           string
	TotalValue     float64
	DroppedValue   float64
	DestroyedValue float64
	// etc.
}

// You can also add your "Params" struct
type Params struct {
	Corporations []int
	Alliances    []int
	Characters   []int
}
