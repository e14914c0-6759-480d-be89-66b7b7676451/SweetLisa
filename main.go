package main

import (
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/pkg/log"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/webserver/router"
)

func main() {
	//go func() {
	//	_, err := bot.New(config.GetConfig().BotToken, nil)
	//	if err != nil {
	//		log.Fatal("Bot: %v", err)
	//	}
	//}()
	log.Fatal("%v", router.Run())
}
