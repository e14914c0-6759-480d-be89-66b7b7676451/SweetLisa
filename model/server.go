package model

import (
	"crypto/sha1"
	"fmt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/common"
	"reflect"
	"strings"
	"time"
)

const (
	BucketServer    = "server"
	MaxFailureCount = 10
)

type Protocol string

const (
	ProtocolVMessTCP    Protocol = "vmess"
	ProtocolShadowsocks          = "shadowsocks"
)

func (p Protocol) Valid() bool {
	switch p {
	case ProtocolVMessTCP, ProtocolShadowsocks:
		return true
	default:
		return false
	}
}

type Server struct {
	// Every server should have a server ticket, which should be included in each API interactions
	Ticket string
	// Name is also the proxy node name
	Name string
	// Hosts can be IPs and domains (split by ",")
	Hosts string `json:"Host"`
	// Port is shared by management and proxy
	Port int
	// Argument is used to connect and manage the server
	Argument Argument
	// NetType indicates if the server supports v4(b01), v6(b10) or both(b11)
	NetType uint8

	// FailureCount is the number of consecutive failed pings
	FailureCount int
	// LastSeen is the time of last succeed ping
	LastSeen time.Time
	// SyncNextSeen is a flag indicates the server should be sync next seen
	SyncNextSeen bool
}

func GetFirstHost(host string) string {
	return strings.SplitN(host, ",", 2)[0]
}

func GetUserArgument(serverTicket, userTicket string) Argument {
	h := sha1.New()
	h.Write([]byte(serverTicket))
	h.Write([]byte(userTicket))
	b := h.Sum(nil)
	return Argument{
		Protocol: ProtocolShadowsocks,
		Password: common.Base62Encoder.Encode(b)[:21],
		Method:   "chacha20-ietf-poly1305",
	}
}

func GetRelayUserArgument(serverTicket, relayTicket, userTicket string) Argument {
	h := sha1.New()
	h.Write([]byte(serverTicket))
	h.Write([]byte(relayTicket))
	h.Write([]byte(userTicket))
	b := h.Sum(nil)
	return Argument{
		Protocol: ProtocolShadowsocks,
		Password: common.Base62Encoder.Encode(b)[:21],
		Method:   "chacha20-ietf-poly1305",
	}
}

type Argument struct {
	// Required
	Protocol Protocol `json:",omitempty"`
	// Optional
	Username string `json:",omitempty"`
	// Required
	Password string `json:",omitempty"`
	// Optional
	Method string `json:",omitempty"`
}

func (a Argument) Hash() string {
	h := sha1.New()
	v := reflect.ValueOf(a)
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		h.Write([]byte(fmt.Sprint(field.Interface())))
	}
	return common.Base95Encoder.Encode(h.Sum(nil))
}
