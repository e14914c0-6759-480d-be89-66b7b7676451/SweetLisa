package model

import (
	"fmt"
	"time"
)

const BucketVote = "vote"

var VoteExpiredErr = fmt.Errorf("vote expired")

type Vote struct {
	ExpireAt       time.Time
	ChatIdentifier string
	Number         int
}

func init() {
	go ExpireCleanBackground(BucketVerification, 60*time.Second, func(v interface{}, now time.Time) (expired bool) {
		return now.After(v.(Verification).ExpireAt)
	})()
}
