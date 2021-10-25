package command_handler

import (
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/bot"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/config"
	tb "gopkg.in/tucnak/telebot.v2"
	"net/url"
	"path"
)

func init() {
	bot.RegisterCommands("sweetlisa", SweetLisa)
}

func SweetLisa(b *bot.Bot, m *tb.Message, params []string) {
	chatIdentifier := b.ChatIdentifier(m.Chat)
	u := url.URL{
		Scheme: "https",
		Host:   config.GetConfig().Host,
		Path:   path.Join("chat", chatIdentifier),
	}
	b.Bot.Reply(m, u.String(), tb.Silent, tb.NoPreview)
}
