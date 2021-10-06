package model

const (
	BucketServer = "server"
)

type ProxyProtocol int

const (
	VMessTCP ProxyProtocol = iota
)

type Server struct {
	// Every server should have a server ticket, which should be included in each API interactions
	Ticket string
	// Name is also the proxy node name
	Name string
	// Host can be either IP or domain
	Host string
	// Port is shared by management and proxy
	Port int
	// Protocol is the proxy protocol combination to use
	Protocol ProxyProtocol
	// Alive will be false if the server has no heart for more than 10 minutes
	Alive bool
}
