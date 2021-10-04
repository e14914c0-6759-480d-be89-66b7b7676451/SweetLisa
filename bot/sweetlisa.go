package bot

import (
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/config"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/pkg/log"
	tb "gopkg.in/tucnak/telebot.v2"
	"net/url"
	"path"
)

func (b *Bot) SweetLisa(m *tb.Message) {
	log.Warn("%v", m.Sender.ID)
	if false {
		_, _ = b.tb.Reply(m, "Please MUST keep anonymous before use it.")
		return
	}
	chatIdentifier := b.ChatIdentifier(m.Chat)
	u := url.URL{
		Scheme: "https",
		Host:   config.GetConfig().Host,
		Path:   path.Join("chat", chatIdentifier),
	}
	b.tb.Reply(m, u.String(), tb.Silent, tb.NoPreview)
}