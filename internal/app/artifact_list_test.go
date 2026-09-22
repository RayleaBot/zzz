package app

import "testing"

func TestArtifactListDropsPiecesWhoseMainStatIsTheirOnlyUsefulStat(t *testing.T) {
	weights := map[string]float64{"atk": 75, "cpct": 100, "cdmg": 100, "dmg": 100}
	piece := func(main string, subs ...string) ScoredPiece {
		p := ScoredPiece{Main: ScoredAttr{Key: main}}
		for _, key := range subs {
			p.Attrs = append(p.Attrs, ScoredAttr{Key: key})
		}
		return p
	}
	if !onlyMainUseful(3, piece("atk", "hpPlus", "def", "mastery", "defPlus"), weights) {
		t.Fatal("a sands with only its main stat useful was kept")
	}
	if onlyMainUseful(3, piece("atk", "hpPlus", "cpct"), weights) || onlyMainUseful(4, piece("hp", "def"), weights) {
		t.Fatal("a piece with a useful substat or a useless main stat was dropped")
	}
	// Pieces with a fixed main stat stay, as upstream means.
	if onlyMainUseful(2, piece("atkPlus", "hpPlus"), map[string]float64{"atkPlus": 50}) || onlyMainUseful(5, piece("cpct"), nil) {
		t.Fatal("fixed main stat or weightless character dropped")
	}
}
