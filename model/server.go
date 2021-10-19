package model

import (
	"crypto/sha1"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/common"
	"github.com/eknkc/basex"
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
}

func (s *Server) GetUserArgument(userTicket string) Argument {
	h := sha1.New()
	h.Write([]byte(s.Ticket))
	h.Write([]byte(userTicket))
	b := h.Sum(nil)
	encoder, _ := basex.NewEncoding(common.Alphabet)
	return Argument{
		Protocol: Shadowsocks,
		Password: encoder.Encode(b)[:21],
		Method:   "chacha20-ietf-poly1305",
	}
}

func (s *Server) GetRelayUserArgument(userTicket string, svr Server) Argument {
	h := sha1.New()
	h.Write([]byte(svr.Ticket))
	h.Write([]byte(s.Ticket))
	h.Write([]byte(userTicket))
	b := h.Sum(nil)
	encoder, _ := basex.NewEncoding(common.Alphabet)
	return Argument{
		Protocol: Shadowsocks,
		Password: encoder.Encode(b)[:21],
		Method:   "chacha20-ietf-poly1305",
	}
}

type Argument struct {
	// Required
	Protocol ProxyProtocol
	// Optional
	Username string
	// Required
	Password string
	// Optional
	Method string
}
