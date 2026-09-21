package main

import (
	"context"
	"fmt"
	"os"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	gamekit "github.com/RayleaBot/game-plugin-kit"
	"github.com/RayleaBot/plugin-zzz/internal/assets"
)

func main() {
	application, err := gamekit.New(assets.Kit(), os.Getenv("RAYLEABOT_PLUGIN_DATA_DIR"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "游戏插件无法启动：%v\n", err)
		os.Exit(1)
	}
	err = rayleabot.Run(context.Background(), rayleabot.Options{}, application)
	application.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "游戏插件会话已结束：%v\n", err)
		os.Exit(1)
	}
}
