package model

import (
	"time"
)

const (
	BucketTicket = "ticket"
	TicketLength = 52
)

type TicketType int

const (
	TicketTypeUser TicketType = iota
	TicketTypeServer
	TicketTypeRelay
	TicketTypeINVALID
)

func (t TicketType) IsValid() bool {
	return t >= 0 && t < TicketTypeINVALID
}

type Ticket struct {
	Ticket         string
	ChatIdentifier string
	Type           TicketType
	ExpireAt       time.Time
}
