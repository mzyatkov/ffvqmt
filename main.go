package main

import (
	"embed"
	"fmt"
	"log"
	"os"

	"github.com/mzyatkov/ffvqmt/internal/cli"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var icon []byte

func main() {
	parsed, err := cli.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "ffvqmt: "+err.Error())
		fmt.Fprintln(os.Stderr, cli.Usage())
		os.Exit(2)
	}
	if parsed.ShowHelp {
		fmt.Println(cli.Usage())
		return
	}
	if parsed.ShowVersion {
		fmt.Println("FFvqmt 1.0.0 (MIT)")
		return
	}

	app := NewApp(parsed)

	err = wails.Run(&options.App{
		Title:  "FFvqmt",
		Width:  1280,
		Height: 800,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.Startup,
		OnShutdown:       app.Shutdown,
		OnDomReady:       app.DomReady,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
		},
		Mac: &mac.Options{
			// Use the default native title bar so the traffic lights sit in
			// their own row above our menu bar — avoids the overlap/alignment
			// issues caused by TitleBarHiddenInset.
			TitleBar:             mac.TitleBarDefault(),
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			About: &mac.AboutInfo{
				Title:   "FFvqmt",
				Message: "FFvqmt — Fast Forward Video Quality Measurement Tool\nOpen-source under the MIT License",
				Icon:    icon,
			},
		},
		Linux: &linux.Options{
			Icon:                icon,
			WindowIsTranslucent: false,
			ProgramName:         "FFvqmt",
		},
	})

	if err != nil {
		log.Fatal(err)
	}

	if parsed.Exit {
		os.Exit(0)
	}
}
