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
	TicketTypeINVALID
)

func (t TicketType) IsValid() bool {
	return t < TicketTypeINVALID
}

type Ticket struct {
	Ticket         string
	ChatIdentifier string
	Type           TicketType
	ExpireAt       time.Time
}
