package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Create application with custom modern options
	err := wails.Run(&options.App{
		Title:             "QBFind",
		Width:             840,
		Height:            540,
		MinWidth:          640,
		MinHeight:         420,
		Frameless:         true, // Allows full custom dark HTML titlebar
		BackgroundColour:  &options.RGBA{R: 0, G: 0, B: 0, A: 255}, // Pitch black background
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
