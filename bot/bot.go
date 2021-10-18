package bot

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
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
	return StringToUUID5(strChatID)
}

// StringToUUID5 is from https://github.com/XTLS/Xray-core/issues/158
func StringToUUID5(str string) string {
	var Nil [16]byte
	h := sha1.New()
	h.Write(Nil[:])
	h.Write([]byte(str))
	u := h.Sum(nil)[:16]
	u[6] = (u[6] & 0x0f) | (5 << 4)
	u[8] = u[8]&(0xff>>2) | (0x02 << 6)
	buf := make([]byte, 36)
	hex.Encode(buf[0:8], u[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], u[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], u[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], u[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:], u[10:])
	return string(buf)
}