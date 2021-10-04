package model

import (
	"fmt"
	"time"
)

const BucketVerification = "verification"

var VerificationExpiredErr = fmt.Errorf("verification expired")

type Verification struct {
	ExpireAt       time.Time
	ChatIdentifier string
	Progress       VerificationProgress
}

type VerificationProgress int

const (
	VerificationWaiting = iota
	VerificationDone
)

func init() {
	go ExpireCleanBackground(BucketVerification, 10*time.Second, func(v interface{}, now time.Time) (expired bool) {
		return now.After(v.(Verification).ExpireAt)
	})()
}
