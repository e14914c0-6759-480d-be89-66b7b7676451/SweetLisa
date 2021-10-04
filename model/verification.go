package model

import (
	"fmt"
	"time"
)

const BucketVerification = "verification"

var VerificationTimeoutErr = fmt.Errorf("verification timeout")

type Verification struct {
	Expire         time.Time
	ChatIdentifier string
	Progress       VerificationProgress
}

type VerificationProgress int

const (
	VerificationWaiting = iota
	VerificationPass
)

func init() {
	go ExpireCleanBackground(BucketVerification, 60*time.Second, func(v interface{}, now time.Time) (expired bool) {
		return now.After(v.(Verification).Expire)
	})()
}
