package model

import (
	"crypto/sha1"
	"encoding/hex"
	"time"
)

const (
	BucketServer = "server"
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
	// ManageArgument is used to connect and manage the server
	ManageArgument Argument
}

func (s *Server) GetUserArgument(userTicket string) Argument {
	h := sha1.New()
	h.Write([]byte(s.Ticket))
	h.Write([]byte(userTicket))
	b := h.Sum(nil)
	return Argument{
		Protocol: Shadowsocks,
		Password: hex.EncodeToString(b)[:21],
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
