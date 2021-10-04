package bot

import (
	"fmt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/common"
	tb "gopkg.in/tucnak/telebot.v2"
	"time"
)

type Bot struct {
	tb *tb.Bot
}

func New(token string, poller *tb.LongPoller) (*Bot, error) {
	if poller == nil {
		poller = &tb.LongPoller{Timeout: 15 * time.Second}
	}
	b, err := tb.NewBot(tb.Settings{
		Token:  token,
		Poller: poller,
	})
	if err != nil {
		return nil, err
	}
	bot := &Bot{
		tb: b,
	}
	b.Handle("/sweetlisa", bot.SweetLisa)
	b.Handle("/verify", bot.Verify)
	b.Start()
	return bot, nil
}

func (b *Bot) ChatIdentifier(c *tb.Chat) string {
	strChatID := fmt.Sprintf("%v", c.ID)
	hash := common.Bytes2Sha256([]byte(strChatID), []byte(b.tb.Token))
	return string(hash[:])
}
