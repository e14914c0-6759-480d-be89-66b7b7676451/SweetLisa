package commandHandler

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
	log.Warn("text", m.Text)
	// m.Text should be a random string for verification
	if err := service.VerificationToPassed(m.Text, chatIdentifier); err != nil {
		b.Bot.Reply(m, err.Error(), tb.Silent, tb.NoPreview)
	} else {
		b.Bot.Reply(m, "pass", tb.Silent, tb.NoPreview)
	}
}
