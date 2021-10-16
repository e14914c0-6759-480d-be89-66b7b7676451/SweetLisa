package model

import (
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/pkg/log"
	jsoniter "github.com/json-iterator/go"
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

func init() {
	go ExpireCleanBackground(BucketVerification, 1*time.Hour, func(b []byte, now time.Time) (expired bool) {
		var ticket Ticket
		err := jsoniter.Unmarshal(b, &ticket)
		if err != nil {
			log.Warn("clean ticket: %v", err)
			return false
		}
		if ticket.ExpireAt.IsZero() {
			// never expire if no expiration time was given
			return false
		}
		return now.After(ticket.ExpireAt)
	})()
}
