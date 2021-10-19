package model

import (
	"github.com/boltdb/bolt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/db"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/pkg/log"
	"sync"
	"time"
)

func ExpireCleanBackground(bucket string, cleanInterval time.Duration, f func(b []byte, now time.Time) (expired bool)) func() {
	return func() {
		tick := time.Tick(cleanInterval)
		for now := range tick {
			if err := db.DB().Update(func(tx *bolt.Tx) error {
				bkt, err := tx.CreateBucketIfNotExists([]byte(bucket))
				if err != nil {
					return err
				}
				var listClean [][]byte
				if err = bkt.ForEach(func(k, b []byte) error {
					if f(b, now) {
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

func TickUpdateBackground(bucket string, interval time.Duration, f func(b []byte, now time.Time) (todo func(b []byte) []byte)) func() {
	return func() {
		type keyTodo struct {
			Key  []byte
			Todo func(b []byte) []byte
		}
		tick := time.Tick(interval)
		for now := range tick {
			go func(now time.Time) {
				// mu projects the listUpdate
				var mu sync.Mutex
				var keysTodo []keyTodo
				var wg sync.WaitGroup
				if err := db.DB().View(func(tx *bolt.Tx) error {
					bkt := tx.Bucket([]byte(bucket))
					if bkt == nil {
						return nil
					}
					if err := bkt.ForEach(func(k, b []byte) error {
						wg.Add(1)
						// k and b have their own lifecycle
						key := make([]byte, len(k))
						val := make([]byte, len(b))
						copy(key, k)
						copy(val, b)
						go func(k, b []byte) {
							defer wg.Done()
							if todo := f(b, now); todo != nil {
								mu.Lock()
								keysTodo = append(keysTodo, keyTodo{Key: k, Todo: todo})
								mu.Unlock()
							}
						}(key, val)
						return nil
					}); err != nil {
						return err
					}
					return nil
				}); err != nil {
					log.Warn("TickUpdateBackground: View bucket %v: %v", bucket, err)
				}
				wg.Wait()
				if len(keysTodo) == 0 {
					return
				}
				if err := db.DB().Update(func(tx *bolt.Tx) error {
					bkt, err := tx.CreateBucketIfNotExists([]byte(bucket))
					if err != nil {
						return err
					}
					for _, k := range keysTodo {
						b := k.Todo(bkt.Get(k.Key))
						if b == nil {
							continue
						}
						if err := bkt.Put(k.Key, b); err != nil {
							log.Warn("TickUpdateBackground: Update bucket %v: %v", bucket, err)
							continue
						}
					}
					return nil
				}); err != nil {
					log.Warn("TickUpdateBackground: Update bucket %v: %v", bucket, err)
				}
			}(now)
		}
	}
}
