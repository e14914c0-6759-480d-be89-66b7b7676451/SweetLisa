package model

import "time"

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

func init() {
	go ExpireCleanBackground(BucketVerification, 1*time.Hour, func(v interface{}, now time.Time) (expired bool) {
		t := v.(Ticket)
		if t.ExpireAt.IsZero() {
			// never expire if no expiration time was given
			return false
		}
		return now.After(t.ExpireAt)
	})()
}
