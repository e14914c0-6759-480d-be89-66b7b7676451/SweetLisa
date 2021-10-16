package model

import (
	"fmt"
	jsoniter "github.com/json-iterator/go"
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

func init() {
	go ExpireCleanBackground(BucketVerification, 10*time.Second, func(b []byte, now time.Time) (expired bool) {
		var v Verification
		if err := jsoniter.Unmarshal(b, &v); err != nil {
			// invalid verifications are regarded as expired
			return true
		}
		return now.After(v.ExpireAt)
	})()
}
