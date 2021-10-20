package model

import (
	"crypto/sha1"
	"fmt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/common"
	"reflect"
	"time"
)

const (
	BucketServer    = "server"
	MaxFailureCount = 10
)

type ProxyProtocol string

const (
	VMessTCP    ProxyProtocol = "vmess"
	Shadowsocks               = "shadowsocks"
)

func (p ProxyProtocol) Valid() bool {
	switch p {
	case VMessTCP, Shadowsocks:
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
	// Host can be either IP or domain
	Host string
	// Port is shared by management and proxy
	Port int
	// FailureCount is the count of failed ping
	FailureCount int
	// LastSeen is the time of last succeed ping
	LastSeen time.Time
	// Argument is used to connect and manage the server
	Argument Argument

	SyncNextSeen bool
}

func GetUserArgument(serverTicket, userTicket string) Argument {
	h := sha1.New()
	h.Write([]byte(serverTicket))
	h.Write([]byte(userTicket))
	b := h.Sum(nil)
	return Argument{
		Protocol: Shadowsocks,
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
		Protocol: Shadowsocks,
		Password: common.Base62Encoder.Encode(b)[:21],
		Method:   "chacha20-ietf-poly1305",
	}
}

type Argument struct {
	// Required
	Protocol ProxyProtocol `json:",omitempty"`
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
