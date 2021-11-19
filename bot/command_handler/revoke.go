package command_handler

import (
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/bot"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/pkg/log"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/service"
	tb "gopkg.in/tucnak/telebot.v2"
)

func init() {
	bot.RegisterCommands("revoke", Revoke)
}

func Revoke(b *bot.Bot, m *tb.Message, params []string) {
	chatIdentifier := b.ChatIdentifier(m.Chat)
	if len(params) < 1 {
		b.Bot.Reply(m, "Invalid revoke params. Format:\n/revoke <your_ticket>", tb.Silent, tb.NoPreview)
		return
	}

	log.Info("Revoke: chatIdentifier: %v, text: %v", chatIdentifier, params[0])
	// m.Text should be a random string for verification
	if err := service.RevokeTicket(nil, params[0], chatIdentifier); err != nil {
		b.Bot.Reply(m, err.Error(), tb.Silent, tb.NoPreview)
	} else {
		b.Bot.Reply(m, "Revoked.", tb.Silent, tb.NoPreview)
	}
}
