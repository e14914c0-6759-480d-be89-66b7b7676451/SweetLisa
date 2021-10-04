package model

import "time"

const (
	BucketTicket = "ticket"
	TicketLength = 52
)

type Ticket struct {
	ChatIdentifier string
	ExpireAt       time.Time
}

func init() {
	go ExpireCleanBackground(BucketVerification, 1*time.Hour, func(v interface{}, now time.Time) (expired bool) {
		return now.After(v.(Ticket).ExpireAt)
	})()
}
