package main

import (
	"embed"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/bot"
	_ "github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/bot/commandHandler"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/config"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/pkg/log"
	_ "github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/pkg/shadowsocks"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/webserver/router"
)

//go:embed static/*
var f embed.FS

func main() {
	GoBackgrounds()
	go SyncAll()
	go func() {
		_, err := bot.New(config.GetConfig().BotToken, nil)
		if err != nil {
			log.Fatal("Bot: %v", err)
		}
	}()
	log.Fatal("%v", router.Run(f))
}
