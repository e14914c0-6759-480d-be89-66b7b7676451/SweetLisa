package model

import (
	"fmt"
	"time"
)

const BucketVerification = "verification"

var VerificationExpiredErr = fmt.Errorf("verification expired")

type Verification struct {
	Code           string
	ExpireAt       time.Time
	ChatIdentifier string
	Progress       VerificationProgress
}

type VerificationProgress int

const (
	VerificationWaiting = iota
	VerificationDone
)