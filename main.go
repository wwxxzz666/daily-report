package main

import (
	"embed"
	"log"

	"daily-report/internal/config"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("!!! 主函数 panic: %v", r)
		}
	}()

	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	if !acquireLock() {
		log.Println("已有实例运行，退出")
		return
	}
	defer releaseLock()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	logFile, err := setupLog(cfg.LogPath())
	if err == nil {
		defer logFile.Close()
	}

	log.Println("日报助手启动中...")

	app := &App{}

	log.Println("调用 wails.Run...")
	err = wails.Run(&options.App{
		Title:  "日报助手",
		Width:  420,
		Height: 700,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:     app.startup,
		OnShutdown:    app.shutdown,
		OnBeforeClose: app.shouldClose,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
		},
	})
	log.Printf("wails.Run 返回, err=%v", err)
	if err != nil {
		log.Fatalf("启动失败: %v", err)
	}

	log.Println("日报助手已退出")
}
