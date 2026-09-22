package app

import (
	"encoding/json"
	"fmt"
)

// GameData is the fixed reference data a game plugin embeds. The optional
// parts are nil when the game has no such feature.
type GameData struct {
	Resources  GameResources
	Simulation *SimulationDeck
	CloudGear  *cloudGearData
	Enemies    *EnemyTable
}

func parseGameData(assets Assets) (*GameData, error) {
	data := &GameData{}
	if err := json.Unmarshal(assets.Resources, &data.Resources); err != nil || data.Resources.Version == "" {
		return nil, fmt.Errorf("materials and banner data are invalid")
	}
	if assets.Simulation != nil {
		deck := &SimulationDeck{}
		if err := json.Unmarshal(assets.Simulation, deck); err != nil {
			return nil, fmt.Errorf("simulation data is invalid")
		}
		for _, items := range [][]string{deck.FiveCharacters, deck.FiveWeapons, deck.FourCharacters, deck.FourWeapons, deck.ThreeWeapons} {
			if len(items) == 0 {
				return nil, fmt.Errorf("simulation data has an empty pool")
			}
		}
		if deck.Version == "" || len(deck.Banners) == 0 {
			return nil, fmt.Errorf("simulation data has no banners")
		}
		data.Simulation = deck
	}
	if assets.CloudPanels != nil {
		gear := &cloudGearData{}
		if err := json.Unmarshal(assets.CloudPanels, gear); err != nil {
			return nil, fmt.Errorf("cloud panel equipment data is invalid")
		}
		data.CloudGear = gear
	}
	if assets.Enemies != nil {
		table := &EnemyTable{}
		if err := json.Unmarshal(assets.Enemies, table); err != nil || len(table.Curves) == 0 {
			return nil, fmt.Errorf("enemy data is invalid")
		}
		data.Enemies = table
	}
	return data, nil
}
