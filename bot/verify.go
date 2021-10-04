package bot

import (
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/pkg/log"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/service"
	tb "gopkg.in/tucnak/telebot.v2"
)

func (b *Bot) Verify(m *tb.Message) {
	log.Warn("%v", m.Sender.ID)
	if false {
		_, _ = b.tb.Reply(m, "Please MUST keep anonymous before use it.")
		return
	}
	chatIdentifier := b.ChatIdentifier(m.Chat)
	log.Warn("text", m.Text)
	// m.Text should be a random string for verification
	if err := service.VerificationDone(m.Text, chatIdentifier); err != nil {
		b.tb.Reply(m, err.Error(), tb.Silent, tb.NoPreview)
	} else {
		b.tb.Reply(m, "pass", tb.Silent, tb.NoPreview)
	}
}
