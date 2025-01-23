package model

import (
	"encoding/json"
	"html/template"
	"time"

	"golang.org/x/oauth2"
)

// If you want a helper for JSON unmarshal:
func JSONUnmarshal(data []byte, out interface{}) error {
	return json.Unmarshal(data, out)
}

// ----------------------------------------------------------------------
// ESI-Specific Data Structures
// ----------------------------------------------------------------------

// EsiCharacter is an EVE Online character as returned by ESI.
type EsiCharacter struct {
	Birthday       time.Time `json:"birthday"`
	BloodlineID    int       `json:"bloodline_id"`
	CorporationID  int       `json:"corporation_id"`
	Description    string    `json:"description"`
	Gender         string    `json:"gender"`
	Name           string    `json:"name"`
	RaceID         int       `json:"race_id"`
	SecurityStatus float64   `json:"security_status"`
}

// EsiAlliance represents an EVE Online alliance from ESI.
type EsiAlliance struct {
	CreatorCorporationID  int       `json:"creator_corporation_id"`
	CreatorID             int       `json:"creator_id"`
	DateFounded           time.Time `json:"date_founded"`
	ExecutorCorporationID int       `json:"executor_corporation_id"`
	Name                  string    `json:"name"`
	Ticker                string    `json:"ticker"`
}

// EsiCorporation is detailed corporation info from ESI.
type EsiCorporation struct {
	AllianceID    int       `json:"alliance_id"`
	CeoID         int       `json:"ceo_id"`
	CreatorID     int       `json:"creator_id"`
	DateFounded   time.Time `json:"date_founded"`
	Description   string    `json:"description"`
	HomeStationID int       `json:"home_station_id"`
	MemberCount   int       `json:"member_count"`
	Name          string    `json:"name"`
	Shares        int       `json:"shares"`
	TaxRate       float64   `json:"tax_rate"`
	Ticker        string    `json:"ticker"`
	URL           string    `json:"url"`
}

// FailedCharacters tracks CharacterIDs that failed (404, etc.).
type FailedCharacters struct {
	CharacterIDs map[int]bool `json:"character_ids"`
}

// CharacterResponse is an ESI response shape for a single character.
type CharacterResponse struct {
	AllianceID     int32     `json:"alliance_id,omitempty"`
	Birthday       time.Time `json:"birthday"`
	BloodlineID    int32     `json:"bloodline_id"`
	CorporationID  int32     `json:"corporation_id"`
	Description    string    `json:"description,omitempty"`
	FactionID      int32     `json:"faction_id,omitempty"`
	Gender         string    `json:"gender"`
	Name           string    `json:"name"`
	RaceID         int32     `json:"race_id"`
	SecurityStatus float64   `json:"security_status,omitempty"`
	Title          string    `json:"title,omitempty"`
}

// EsiCorporationInfo is another shape you had for corporations.
type EsiCorporationInfo struct {
	AllianceID    *int32  `json:"alliance_id,omitempty"`
	CEOId         int32   `json:"ceo_id"`
	CreatorID     int32   `json:"creator_id"`
	DateFounded   *string `json:"date_founded,omitempty"`
	Description   *string `json:"description,omitempty"`
	FactionID     *int32  `json:"faction_id,omitempty"`
	HomeStationID *int32  `json:"home_station_id,omitempty"`
	MemberCount   int32   `json:"member_count"`
	Name          string  `json:"name"`
	Shares        *int64  `json:"shares,omitempty"`
	TaxRate       float64 `json:"tax_rate"`
	Ticker        string  `json:"ticker"`
	URL           *string `json:"url,omitempty"`
	WarEligible   *bool   `json:"war_eligible,omitempty"`
}

// EsiCharacterPortrait holds various portrait sizes for a character.
type EsiCharacterPortrait struct {
	Px128x128 string `json:"px128x128"`
	Px256x256 string `json:"px256x256"`
	Px512x512 string `json:"px512x512"`
	Px64x64   string `json:"px64x64"`
}

// ----------------------------------------------------------------------
// EsiKillMail + typed VictimItem
// ----------------------------------------------------------------------

// EsiKillMail is an ESI structure for killmail details.
type EsiKillMail struct {
	KillMailID    int        `json:"killmail_id"`
	KillMailTime  time.Time  `json:"killmail_time"`
	SolarSystemID int        `json:"solar_system_id"`
	Victim        Victim     `json:"victim"`
	Attackers     []Attacker `json:"attackers"`
}

// Attacker is an ESI shape for a killmail attacker.
type Attacker struct {
	AllianceID     int     `json:"alliance_id"`
	CharacterID    int     `json:"character_id"`
	CorporationID  int     `json:"corporation_id"`
	DamageDone     int     `json:"damage_done"`
	FinalBlow      bool    `json:"final_blow"`
	SecurityStatus float64 `json:"security_status"`
	ShipTypeID     int     `json:"ship_type_id"`
	WeaponTypeID   int     `json:"weapon_type_id"`
}

// Victim is an ESI shape for a killmail victim.
// Items is now typed as []VictimItem (instead of []interface{}).
type Victim struct {
	CharacterID   int          `json:"character_id"`
	CorporationID int          `json:"corporation_id"`
	AllianceID    int          `json:"alliance_id,omitempty"`
	DamageTaken   int          `json:"damage_taken"`
	Items         []VictimItem `json:"items"` // typed sub-items
	Position      struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
		Z float64 `json:"z"`
	} `json:"position"`
	ShipTypeID int `json:"ship_type_id"`
}

// VictimItem is typed so we can do recursion.
type VictimItem struct {
	Flag              int          `json:"flag"`
	ItemTypeID        int          `json:"item_type_id"`
	QuantityDestroyed int64        `json:"quantity_destroyed,omitempty"`
	QuantityDropped   int64        `json:"quantity_dropped,omitempty"`
	Singleton         int          `json:"singleton,omitempty"`
	Items             []VictimItem `json:"items,omitempty"` // Recursively nested items
}

// ----------------------------------------------------------------------
// FlattenedKillMail merges ESI killmail + Zkill data
// ----------------------------------------------------------------------
type FlattenedKillMail struct {
	KillMailID    int64     `json:"killmail_id"`
	KillMailTime  time.Time `json:"killmail_time"`
	SolarSystemID int       `json:"solar_system_id"`
	Victim        Victim    `json:"victim"`
	Attackers     []Attacker

	// zKill extra fields:
	LocationID     int64   `json:"locationID"`
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

// ConvertToFlattened merges an EsiKillMail with a ZkillMail into a FlattenedKillMail.
func ConvertToFlattened(esi EsiKillMail, zkill ZkillMail) FlattenedKillMail {
	return FlattenedKillMail{
		KillMailID:     int64(esi.KillMailID),
		KillMailTime:   esi.KillMailTime,
		SolarSystemID:  esi.SolarSystemID,
		Victim:         esi.Victim,
		Attackers:      esi.Attackers,
		LocationID:     zkill.ZKB.LocationID,
		Hash:           zkill.ZKB.Hash,
		FittedValue:    zkill.ZKB.FittedValue,
		DroppedValue:   zkill.ZKB.DroppedValue,
		DestroyedValue: zkill.ZKB.DestroyedValue,
		TotalValue:     zkill.ZKB.TotalValue,
		Points:         zkill.ZKB.Points,
		NPC:            zkill.ZKB.NPC,
		Solo:           zkill.ZKB.Solo,
		Awox:           zkill.ZKB.Awox,
	}
}

// ZkillMailFeedResponse is for zKill’s streaming feed
type ZkillMailFeedResponse struct {
	KillmailID    int64      `json:"killmail_id"`
	SolarSystemID int        `json:"solar_system_id"`
	Victim        Victim     `json:"victim"`
	Attackers     []Attacker `json:"attackers"`
	ZKB           ZKB        `json:"zkb"`
}

// ----------------------------------------------------------------------
// ZKill-Specific data
// ----------------------------------------------------------------------
type ZkillMail struct {
	KillMailID int64 `json:"killmail_id"`
	ZKB        ZKB   `json:"zkb"`
}

type ZKB struct {
	LocationID     int64   `json:"locationID"`
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

// ----------------------------------------------------------------------
// Corporation, Alliance, Character, etc.
// ----------------------------------------------------------------------
type Corporation struct {
	AllianceID    *int32  `json:"alliance_id,omitempty"`
	CEOId         int32   `json:"ceo_id"`
	CreatorID     int32   `json:"creator_id"`
	DateFounded   *string `json:"date_founded,omitempty"`
	Description   *string `json:"description,omitempty"`
	FactionID     *int32  `json:"faction_id,omitempty"`
	HomeStationID *int32  `json:"home_station_id,omitempty"`
	MemberCount   int32   `json:"member_count"`
	Name          string  `json:"name"`
	Shares        *int64  `json:"shares,omitempty"`
	TaxRate       float64 `json:"tax_rate"`
	Ticker        string  `json:"ticker"`
	URL           *string `json:"url,omitempty"`
	WarEligible   *bool   `json:"war_eligible,omitempty"`
}

type Alliance struct {
	CreatorCorporationID  int       `json:"creator_corporation_id"`
	CreatorID             int       `json:"creator_id"`
	DateFounded           time.Time `json:"date_founded"`
	ExecutorCorporationID int       `json:"executor_corporation_id"`
	Name                  string    `json:"name"`
	Ticker                string    `json:"ticker"`
}

// ----------------------------------------------------------------------
// Additional Data Structures for "Charts" or "Params"
// ----------------------------------------------------------------------

// ESIData might store loaded alliance/corp/character info in memory.
type ESIData struct {
	AllianceInfos    map[int]EsiAlliance
	CharacterInfos   map[int]EsiCharacter
	CorporationInfos map[int]EsiCorporation
}

type Params struct {
	Corporations []int
	Alliances    []int
	Characters   []int
	Year         int
	EsiData      *ESIData
	ChangedIDs   bool
	NewIDs       *Ids
}

type Ids struct {
	AllianceIDs    []int `json:"alliance_ids"`
	CharacterIDs   []int `json:"character_ids"`
	CorporationIDs []int `json:"corporation_ids"`
}

type ChartData struct {
	KillMails []FlattenedKillMail
	ESIData
	TrackedCharacters []int
	LookupFunc        func(int) string
}

// Chart is a single chart definition with data prep logic, for front-end rendering.
type Chart struct {
	FieldPrefix string
	PrepareFunc func(*ChartData) interface{}
	Description string
	Type        string // e.g. "bar", "line"
}

// TimeFrameData for a specific timeframe in a UI
type TimeFrameData struct {
	Name   string       // e.g. "MTD", "YTD"
	Charts []ChartEntry // Slice of charts
}

// ChartEntry is one chart’s data
type ChartEntry struct {
	Name string      // e.g. "Damage by Character"
	ID   string      // e.g. "damageChart_MTD"
	Data template.JS // JSON for rendering
	Type string      // e.g. "bar", "line"
}

// TemplateData is for passing multiple time frames to a template.
type TemplateData struct {
	TimeFrames []TimeFrameData
}

// ----------------------------------------------------------------------
// Identity / Auth Structures
// ----------------------------------------------------------------------
type Identities struct {
	MainIdentity string                  `json:"main_identity"`
	Tokens       map[string]oauth2.Token `json:"identities"`
}

type AuthState struct {
	Mode      string `json:"mode"`
	AppID     string `json:"app_id"`
	Timestamp int64  `json:"timestamp"`
}

// EncodeState encodes the AuthState to JSON string
func EncodeState(state AuthState) (string, error) {
	b, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// DecodeState decodes a JSON string into an AuthState
func DecodeState(stateStr string) (AuthState, error) {
	var s AuthState
	err := json.Unmarshal([]byte(stateStr), &s)
	return s, err
}

// ----------------------------------------------------------------------
// Additional "config" data or user definitions
// ----------------------------------------------------------------------

type ConfigCharacter struct {
	User
	Location       int64   `json:"Location"`
	HomeLocation   int64   `json:"HomeLocation"`
	CorporationID  int64   `json:"CorporationID"`
	CloneLocations []int64 `json:"CloneLocations"`
	CharacterRoles
	StashList []Stash `json:"StashList"`
}

type CharacterData struct {
	Token     oauth2.Token
	Character ConfigCharacter
}

type CharacterRoles struct {
	Roles        []string `json:"roles"`
	RolesAtBase  []string `json:"roles_at_base"`
	RolesAtHQ    []string `json:"roles_at_hq"`
	RolesAtOther []string `json:"roles_at_other"`
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

// We can define an interface for anything that has a "GetName" if needed.
type Namer interface {
	GetName() string
}

type User struct {
	CharacterID   int64  `json:"CharacterID"`
	CharacterName string `json:"CharacterName"`
}

type Character struct {
	Birthday       time.Time `json:"birthday"`
	BloodlineID    int       `json:"bloodline_id"`
	CorporationID  int       `json:"corporation_id"`
	Description    string    `json:"description"`
	Gender         string    `json:"gender"`
	Name           string    `json:"name"`
	RaceID         int       `json:"race_id"`
	SecurityStatus float64   `json:"security_status"`
}
