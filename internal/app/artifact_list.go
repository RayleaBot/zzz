package app

import (
	"cmp"
	"context"
	"slices"
	"strconv"
	"sync"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// ListedPiece is one piece of equipment on 圣遗物列表 or 遗器列表, with the
// scored panel it is worn on.
type ListedPiece struct {
	Panel     CharacterPanel
	Equipment PanelEquipment
	Scored    ScoredPiece
	Score     float64
}

// ArtifactListImage is 圣遗物列表 or 遗器列表 for a UID.
type ArtifactListImage struct {
	UID    string
	Pieces []ListedPiece
}

type ArtifactListImageBuilder func(ImageContext, ArtifactListImage) (Image, bool)

// artifactListLength is miao's default artisNumber.
const artifactListLength = 28

// artifactList is miao's profileArtisList: every piece on the UID's kept
// panels scored with its character's rule, without the pieces whose main
// stat is their only useful stat, best first.
func (a *App) artifactList(ctx context.Context, event *rayleabot.EventContext, args []string) error {
	uid := ""
	if len(args) > 0 {
		uid = args[0]
	}
	owner, err := a.panelOwner(ctx, event, uid)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	saved, err := a.Profiles.Read(owner.UID)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	var mu sync.Mutex
	var group sync.WaitGroup
	slots := make(chan struct{}, 4)
	pieces := []ListedPiece{}
	for _, kept := range saved.Panels {
		group.Add(1)
		slots <- struct{}{}
		go func() {
			defer func() { <-slots; group.Done() }()
			scored, err := a.scorePanel(ctx, kept.panel())
			if err != nil || scored.ScoreDetail == nil {
				return
			}
			for _, equipment := range scored.Equipment {
				piece, ok := scored.ScoreDetail.Pieces[equipment.Slot]
				if !ok || equipment.Score == nil || onlyMainUseful(equipment.Slot, piece, scored.ScoreDetail.Weights) {
					continue
				}
				mu.Lock()
				pieces = append(pieces, ListedPiece{Panel: scored, Equipment: equipment, Scored: piece, Score: equipment.Score.Value})
				mu.Unlock()
			}
		}()
	}
	group.Wait()
	name := map[string]string{"starrail": "遗器"}[a.Game.ID]
	if name == "" {
		name = "圣遗物"
	}
	if len(pieces) == 0 {
		return event.SendText("请先获取角色面板数据后再查看" + name + "列表...")
	}
	slices.SortFunc(pieces, func(x, y ListedPiece) int {
		return cmp.Or(cmp.Compare(y.Score, x.Score), cmp.Compare(x.Panel.ID, y.Panel.ID), cmp.Compare(x.Equipment.Slot, y.Equipment.Slot))
	})
	pieces = pieces[:min(artifactListLength, len(pieces))]
	view := View{Title: a.Game.Name + name + "列表", Subtitle: "UID " + owner.UID, Rows: []Row{}}
	for _, piece := range pieces {
		view.Rows = append(view.Rows, Row{Label: piece.Panel.Name + " · " + piece.Equipment.Name, Value: piece.Scored.Mark + "分 - " + piece.Scored.Grade + " · +" + strconv.Itoa(piece.Equipment.Level)})
	}
	if a.artifactListImage != nil {
		if drawn, ok := a.artifactListImage(a.imageContext(ctx), ArtifactListImage{UID: owner.UID, Pieces: pieces}); ok {
			view.Image = &drawn
		}
	}
	return a.sendView(ctx, event, view)
}

// onlyMainUseful is miao's filterSingleEffArtis: a piece whose main stat is
// the only stat the character weighs. Pieces with a fixed main stat stay;
// upstream compares the slot key as text and so only keeps the first slot,
// though it means both.
func onlyMainUseful(slot int, piece ScoredPiece, weights map[string]float64) bool {
	if slot <= 2 {
		return false
	}
	useful := func(key string) bool { return weights[key] > 0 }
	if len(weights) == 0 || !useful(piece.Main.Key) {
		return false
	}
	for _, attr := range piece.Attrs {
		if useful(attr.Key) {
			return false
		}
	}
	return true
}
