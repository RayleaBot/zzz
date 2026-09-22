package app

import "testing"

func TestShapeQueryResultKeepsWhatWasAsked(t *testing.T) {
	panel := QueryResult{Data: map[string]any{"avatar_list": []any{map[string]any{"id": 1001.0}, map[string]any{"id": 1002.0}}, "property_info": map[string]any{"5": map[string]any{"name": "暴击率"}}}}
	if err := shapeQueryResult("starrail.character", map[string]any{"character_ids": []any{"1002"}}, &panel); err != nil {
		t.Fatal(err)
	}
	list := panel.Data["avatar_list"].([]any)
	if len(list) != 1 || asText(asObject(list[0])["id"]) != "1002" || panel.Data["property_info"] == nil {
		t.Fatal("wrong selection or lost metadata")
	}
	malformed := QueryResult{Data: map[string]any{"avatar_list": "unexpected"}}
	if err := shapeQueryResult("starrail.character", map[string]any{"character_ids": []any{"1001"}}, &malformed); err == nil {
		t.Fatal("malformed character list accepted")
	}
	for period, dropped := range map[string]string{"1": "last_record", "2": "current_record"} {
		rogue := QueryResult{Data: map[string]any{"current_record": 1, "last_record": 2}}
		if err := shapeQueryResult("starrail.rogue", map[string]any{"schedule_type": period}, &rogue); err != nil || rogue.Data[dropped] != nil || len(rogue.Data) != 1 {
			t.Fatalf("period %s kept %v", period, rogue.Data)
		}
	}
}
