package app

import (
	"encoding/json"
	"fmt"
)

// GameData is the fixed reference data the plugin embeds.
type GameData struct {
	Resources GameResources
}

func parseGameData(assets Assets) (*GameData, error) {
	data := &GameData{}
	if err := json.Unmarshal(assets.Resources, &data.Resources); err != nil || data.Resources.Version == "" {
		return nil, fmt.Errorf("materials and banner data are invalid")
	}
	return data, nil
}
