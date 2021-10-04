package model

import "time"

const BucketSig = "sig"

type Sig struct {
	Sig    string
	Expire time.Time
}

func init() {
	go ExpireCleanBackground(BucketVerification, 1*time.Hour, func(v interface{}, now time.Time) (expired bool) {
		return now.After(v.(Sig).Expire)
	})()
}
