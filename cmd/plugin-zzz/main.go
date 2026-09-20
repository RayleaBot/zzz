package main

import (
	"context"
	"encoding/json"
	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	gamekit "github.com/RayleaBot/game-plugin-kit"
	"github.com/RayleaBot/plugin-zzz/internal/assets"
	"os"
)

func main() {
	var game gamekit.Game
	if json.Unmarshal(assets.Game, &game) != nil {
		os.Exit(1)
	}
	application, err := gamekit.New(game, assets.Catalog, os.Getenv("RAYLEABOT_PLUGIN_DATA_DIR"))
	if err != nil {
		_, _ = os.Stderr.WriteString("游戏插件资料或数据目录不可用。\n")
		os.Exit(1)
	}
	if err = application.SetManifest(assets.Manifest); err != nil {
		os.Exit(1)
	}
	err = rayleabot.Run(context.Background(), rayleabot.Options{}, application)
	application.Close()
	if err != nil {
		_, _ = os.Stderr.WriteString("游戏插件会话已结束。\n")
		os.Exit(1)
	}
}
