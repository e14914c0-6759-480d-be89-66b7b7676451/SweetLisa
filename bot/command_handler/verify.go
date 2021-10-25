package command_handler

import (
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/bot"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/pkg/log"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/service"
	tb "gopkg.in/tucnak/telebot.v2"
)

func init() {
	bot.RegisterCommands("verify", Verify)
}

func Verify(b *bot.Bot, m *tb.Message, params []string) {
	chatIdentifier := b.ChatIdentifier(m.Chat)
	if len(params) < 1 {
		b.Bot.Reply(m, "invalid verify params", tb.Silent, tb.NoPreview)
		return
	}

	log.Info("Verify: chatIdentifier: %v, text: %v", chatIdentifier, params[0])
	// m.Text should be a random string for verification
	if err := service.Verify(nil, params[0], chatIdentifier); err != nil {
		b.Bot.Reply(m, err.Error(), tb.Silent, tb.NoPreview)
	} else {
		b.Bot.Reply(m, "Passed. This code is valid within 2 minutes.", tb.Silent, tb.NoPreview)
	}
}
