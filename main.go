package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Wails uses Go's `embed` package to embed the frontend files into the binary.
// Any files in the frontend/dist folder will be embedded into the binary and
// made available to the frontend.
// See https://pkg.go.dev/embed for more information.

//go:embed all:frontend/dist
var assets embed.FS

func init() {
	// Register the custom events emitted by the SSH service so the binding
	// generator can expose them with strong types to the frontend.
	application.RegisterEvent[SSHOutput]("ssh:output")
	application.RegisterEvent[SSHClosed]("ssh:closed")
	application.RegisterEvent[UploadProgress]("ssh:upload")
	application.RegisterEvent[UploadSummary]("ssh:upload:summary")
}

// main function serves as the application's entry point.
func main() {
	sshServiceRef = newSSHService()

	app := application.New(application.Options{
		Name:        "OpenDesk",
		Description: "A Windows-like desktop for remote Linux hosts over SSH",
		Services: []application.Service{
			application.NewService(sshServiceRef),
		},
		Assets: application.AssetOptions{
			// /preview 用于在线播放远程视频、查看文档；其余走静态资源。
			Handler: previewAssetHandler(application.AssetFileServerFS(assets)),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "OpenDesk",
		Width:            1360,
		Height:           860,
		MinWidth:         1024,
		MinHeight:        640,
		BackgroundColour: application.NewRGB(18, 22, 30),
		URL:              "/",
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
	})

	err := app.Run()
	if err != nil {
		log.Fatal(err)
	}
}
