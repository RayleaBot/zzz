package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-zzz/internal/app"
	"github.com/RayleaBot/plugin-zzz/internal/assets"
)

func main() {
	application, err := app.New(assets.Load(), os.Getenv("RAYLEABOT_PLUGIN_DATA_DIR"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "游戏插件无法启动：%v\n", err)
		os.Exit(1)
	}
	// The package ships the upstream images the templates use.
	if dir := os.Getenv("RAYLEABOT_PLUGIN_PACKAGE_DIR"); dir != "" {
		application.Artwork.Package = filepath.Join(dir, "assets")
	}
	err = rayleabot.Run(context.Background(), rayleabot.Options{}, application)
	application.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "游戏插件会话已结束：%v\n", err)
		os.Exit(1)
	}
}
