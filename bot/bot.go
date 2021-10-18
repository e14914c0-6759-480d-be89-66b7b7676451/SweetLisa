package bot

import (
	"fmt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/common"
	tb "gopkg.in/tucnak/telebot.v2"
	"strings"
	"time"
)

type Bot struct {
	Bot *tb.Bot
}

type CommandHandler func(b *Bot, m *tb.Message, params []string)

var GlobalCommandMapper = make(map[string]CommandHandler)

func RegisterCommands(command string, f CommandHandler) {
	GlobalCommandMapper[command] = f
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
		Bot: b,
	}
	b.Handle(tb.OnChannelPost, func(m *tb.Message) {
		if !strings.HasPrefix(m.Text, "/") || len(m.Text) <= 1 {
			return
		}
		text := strings.TrimPrefix(m.Text, "/")
		fields := strings.Fields(text)
		if handler, ok := GlobalCommandMapper[fields[0]]; ok {
			if !m.FromChannel() || m.Signature != "" {
				_, _ = b.Reply(m, "Please use me from an anonymous channel.")
				return
			}
			handler(bot, m, fields[1:])
		}
	})
	b.Start()
	return bot, nil
}

func (b *Bot) ChatIdentifier(c *tb.Chat) string {
	strChatID := fmt.Sprintf("%v", c.ID)
	return common.StringToUUID5(strChatID)
}
