package app

import (
	"context"
	"strconv"
)

func queryOperationName(operation string, input map[string]any) string {
	if operation == "starrail.character" && input["character_ids"] == nil {
		return "starrail.characters"
	}
	return operation
}

func (a *App) queryScoredPanel(ctx context.Context, client AccountsClient, choice Selection, characterID string) (CharacterPanel, error) {
	panel, err := a.queryCharacterPanel(ctx, client, choice, characterID)
	if err != nil {
		return CharacterPanel{}, err
	}
	return a.scorePanel(ctx, panel)
}

func (a *App) queryCharacterPanel(ctx context.Context, client AccountsClient, choice Selection, characterID string) (CharacterPanel, error) {
	id, err := strconv.Atoi(characterID)
	if err != nil || id <= 0 || id > 1000000000 {
		return CharacterPanel{}, gameError("input_invalid", "请选择有效角色。")
	}
	key := "character_ids"
	if a.Game.ID == "zzz" {
		key = "id_list"
	}
	result, err := client.Execute(ctx, choice, a.Game.ID+".character", map[string]any{key: []any{characterID}})
	if err != nil {
		return CharacterPanel{}, err
	}
	for _, panel := range NormalizePanels(a.Game.ID, result, a.Catalog) {
		if panel.ID == characterID {
			return panel, nil
		}
	}
	return CharacterPanel{}, gameError("character_missing", "此账号未返回所选角色。")
}
