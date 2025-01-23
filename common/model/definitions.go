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
// ESI-Specific Data Structures (originally from your first big snippet)
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

// FailedCharacters tracks CharacterIDs that failed (e.g. 404) so we can skip them.
type FailedCharacters struct {
	CharacterIDs map[int]bool `json:"character_ids"`
}

// CharacterResponse is an ESI response shape for a single character (lightweight).
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

// EsiCorporationInfo is another shape you had for corporations
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
type Victim struct {
	CharacterID   int           `json:"character_id"`
	CorporationID int           `json:"corporation_id"`
	DamageTaken   int           `json:"damage_taken"`
	Items         []interface{} `json:"items"`
	Position      struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
		Z float64 `json:"z"`
	} `json:"position"`
	ShipTypeID int `json:"ship_type_id"`
}

// FlattenedKillMail merges ESI killmail + Zkill data into one struct.
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

type ZkillMailFeedResponse struct {
	KillmailID    int64      `json:"killmail_id"`
	SolarSystemID int        `json:"solar_system_id"`
	Victim        Victim     `json:"victim"`
	Attackers     []Attacker `json:"attackers"`
	ZKB           ZKB        `json:"zkb"`
}

type Corporation struct {
	AllianceID    *int32  `json:"alliance_id,omitempty"`     // CorporationID of the alliance, if any
	CEOId         int32   `json:"ceo_id"`                    // CEO CorporationID (required)
	CreatorID     int32   `json:"creator_id"`                // Creator CorporationID (required)
	DateFounded   *string `json:"date_founded,omitempty"`    // Date the corporation was founded
	Description   *string `json:"description,omitempty"`     // CorporationID description
	FactionID     *int32  `json:"faction_id,omitempty"`      // Faction CorporationID, if any
	HomeStationID *int32  `json:"home_station_id,omitempty"` // Home station CorporationID, if any
	MemberCount   int32   `json:"member_count"`              // Number of members (required)
	Name          string  `json:"name"`                      // Full name of the corporation (required)
	Shares        *int64  `json:"shares,omitempty"`          // Number of shares, if any
	TaxRate       float64 `json:"tax_rate"`                  // Tax rate (required, float with max 1.0 and min 0.0)
	Ticker        string  `json:"ticker"`                    // Short name of the corporation (required)
	URL           *string `json:"url,omitempty"`             // CorporationID URL, if any
	WarEligible   *bool   `json:"war_eligible,omitempty"`    // War eligibility, if any
}

type Alliance struct {
	CreatorCorporationID  int       `json:"creator_corporation_id"`
	CreatorID             int       `json:"creator_id"`
	DateFounded           time.Time `json:"date_founded"`
	ExecutorCorporationID int       `json:"executor_corporation_id"`
	Name                  string    `json:"name"`
	Ticker                string    `json:"ticker"`
}

type CharacterPortrait struct {
	Px128x128 string `json:"px128x128"`
	Px256x256 string `json:"px256x256"`
	Px512x512 string `json:"px512x512"`
	Px64x64   string `json:"px64x64"`
}

// ----------------------------------------------------------------------
// ZKill-Specific Data Structures
// ----------------------------------------------------------------------

// ZkillMail is the structure from zKillboard’s API (partial).
type ZkillMail struct {
	KillMailID int64 `json:"killmail_id"`
	ZKB        ZKB   `json:"zkb"`
}

// ZKB holds zKill’s additional info about values, hash, etc.
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
// Additional Data Structures for "Charts" or "Params"
// ----------------------------------------------------------------------

// ESIData is a structure that might hold loaded alliance/corp/character info in memory.
type ESIData struct {
	AllianceInfos    map[int]EsiAlliance
	CharacterInfos   map[int]EsiCharacter
	CorporationInfos map[int]EsiCorporation
}

// Params for killmail retrieval or other usage.
type Params struct {
	Corporations []int
	Alliances    []int
	Characters   []int
	Year         int
	EsiData      *ESIData
	ChangedIDs   bool
	NewIDs       *Ids
}

// Ids might store new alliance/character/corp IDs discovered?
type Ids struct {
	AllianceIDs    []int `json:"alliance_ids"`
	CharacterIDs   []int `json:"character_ids"`
	CorporationIDs []int `json:"corporation_ids"`
}

// FlattenedKillMail chart usage, etc.
type ChartData struct {
	KillMails []FlattenedKillMail
	ESIData
	TrackedCharacters []int
	LookupFunc        func(int) string
}

// Chart represents a single chart definition with data prep logic.
type Chart struct {
	FieldPrefix string
	PrepareFunc func(*ChartData) interface{}
	Description string
	Type        string // e.g. "bar", "line"
}

// TimeFrameData for a specific time frame in a UI
type TimeFrameData struct {
	Name   string       // e.g. "MTD", "YTD"
	Charts []ChartEntry // Slice of charts
}

// ChartEntry is a single chart’s data
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

// Identities can store multiple tokens keyed by character ID or name
type Identities struct {
	MainIdentity string                  `json:"main_identity"`
	Tokens       map[string]oauth2.Token `json:"identities"`
}

// AuthState is an example structure for OAuth state param usage.
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
// Additional "config" / second snippet data structures
// (Renamed to avoid collisions with the above EsiCharacter, etc.)
// ----------------------------------------------------------------------

// ConfigCharacter is the user-based Character structure that wraps a User plus location info.
type ConfigCharacter struct {
	User
	Location       int64   `json:"Location"`
	HomeLocation   int64   `json:"HomeLocation"`
	CorporationID  int64   `json:"CorporationID"`
	CloneLocations []int64 `json:"CloneLocations"`
	CharacterRoles
	StashList []Stash `json:"StashList"`
}

// CharacterData is a direct pairing of an oauth2.Token with a ConfigCharacter.
type CharacterData struct {
	Token     oauth2.Token
	Character ConfigCharacter
}

// CharacterRoles as used in the config snippet
type CharacterRoles struct {
	Roles        []string `json:"roles"`
	RolesAtBase  []string `json:"roles_at_base"`
	RolesAtHQ    []string `json:"roles_at_hq"`
	RolesAtOther []string `json:"roles_at_other"`
}

// CharacterLocation indicates which system/structure a character is in.
type CharacterLocation struct {
	SolarSystemID int64 `json:"solar_system_id"`
	StructureID   int64 `json:"structure_id"`
}

// CloneLocation for clones/jump clones
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

// Station (public station data)
type Station struct {
	SystemID int64  `json:"system_id"`
	ID       int64  `json:"station_id"`
	Name     string `json:"station_name"`
}

// Structure (for POSes, Upwell structures, etc.)
type Structure struct {
	Name     string `json:"name"`
	OwnerID  int64  `json:"owner_id"`
	SystemID int64  `json:"solar_system_id"`
	TypeID   int64  `json:"type_id"`
}

// Asset references items in a location (type ID, quantity, etc.)
type Asset struct {
	TypeID       int64  `json:"type_id"`
	Quantity     int    `json:"quantity"`
	LocationFlag string `json:"location_flag"`
	LocationType string `json:"location_type"`
	LocationID   int64  `json:"location_id"`
}

// Item is a simpler name/qty structure
type Item struct {
	ID   int64  `json:"item_id"`
	Name string `json:"item_name"`
	Qty  int    `json:"item_qty"`
}

// LocationInventory groups items found at a single location
type LocationInventory struct {
	CharacterID int64          `json:"Id"`
	LocFlag     string         `json:"LocFlag"`
	LocType     string         `json:"LocType"`
	LocID       int            `json:"LocID"`
	Items       map[string]int `json:"Items"`
}

// Stash is a collection of items in a specific system
type Stash struct {
	SystemId   int64  `json:"system_id"`
	SystemName string `json:"system_name"`
	Inventory  []Item `json:"inventory"`
}

// ----------------------------------------------------------------------
// Additional Helpers
// ----------------------------------------------------------------------

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
