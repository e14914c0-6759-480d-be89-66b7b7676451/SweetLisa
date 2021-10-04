package model

import (
	"github.com/boltdb/bolt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/db"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/pkg/log"
	jsoniter "github.com/json-iterator/go"
	"time"
)

func ExpireCleanBackground(bucket string, cleanInterval time.Duration, f func(v interface{}, now time.Time) (expired bool)) func() {
	return func() {
		tick := time.Tick(cleanInterval)
		for range tick {
			now := time.Now()
			if err := db.DB().Update(func(tx *bolt.Tx) error {
				bkt, err := tx.CreateBucketIfNotExists([]byte(bucket))
				if err != nil {
					return err
				}
				var listClean [][]byte
				if err = bkt.ForEach(func(k, b []byte) error {
					var v interface{}
					if err := jsoniter.Unmarshal(b, &v); err != nil {
						return err
					}
					if f(v, now) {
						listClean = append(listClean, k)
					}
					return nil
				}); err != nil {
					return err
				}
				for _, k := range listClean {
					if err = bkt.Delete(k); err != nil {
						return err
					}
				}
				return nil
			}); err != nil {
				log.Warn("Clean bucket %v: %v", bucket, err)
			}
		}
	}
}
