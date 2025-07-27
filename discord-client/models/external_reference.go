package models

type ExternalSystem string

const (
	SystemLeagueOfLegends ExternalSystem = "league_of_legends"
	SystemTFT             ExternalSystem = "teamfight_tactics"
)

type ExternalReference struct {
	System ExternalSystem `json:"system"`
	ID     string         `json:"id"`
}

func (e ExternalReference) IsValid() bool {
	return e.System != "" && e.ID != ""
}