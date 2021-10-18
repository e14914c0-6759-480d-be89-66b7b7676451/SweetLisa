package model

import (
	"context"
	"github.com/boltdb/bolt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/common"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/db"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/pkg/log"
	jsoniter "github.com/json-iterator/go"
	"strconv"
	"sync"
	"time"
)

func GoBackgrounds() {
	// remove expired verifications
	go ExpireCleanBackground(BucketVerification, 10*time.Second, func(b []byte, now time.Time) (expired bool) {
		var v Verification
		if err := jsoniter.Unmarshal(b, &v); err != nil {
			// invalid verifications are regarded as expired
			return true
		}
		return common.Expired(v.ExpireAt)
	})()

	// remove expired tickets
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
		return common.Expired(ticket.ExpireAt)
	})()

	// remove servers that do not be seen 35 days
	go ExpireCleanBackground(BucketServer, 6*time.Hour, func(b []byte, now time.Time) (expired bool) {
		var server Server
		if err := jsoniter.Unmarshal(b, &server); err != nil {
			return false
		}
		if server.FailureCount >= MaxFailureCount && time.Since(server.LastSeen).Hours() > 24*35 {
			// remove corresponding ticket
			if err := db.DB().Update(func(tx *bolt.Tx) error {
				bkt := tx.Bucket([]byte(BucketTicket))
				if bkt == nil {
					return bolt.ErrBucketNotFound
				}
				if err := bkt.Delete([]byte(server.Ticket)); err != nil {
					return err
				}
				return nil
			}); err != nil {
				log.Warn("remove expired server (%v) ticket (%v) fail: %v", server.Name, server.Ticket, err)
				// do not remove this server to avoid inconsistent data
				return false
			} else {
				return true
			}
		} else {
			return false
		}
	})

	// ping at intervals
	go TickUpdateBackground(BucketServer, 60*time.Second, func(b []byte, now time.Time) (updated []byte) {
		var server Server
		if err := jsoniter.Unmarshal(b, &server); err != nil {
			return nil
		}
		if server.FailureCount >= MaxFailureCount {
			// stop the ping and wait for the register
			return nil
		}
		var err error
		defer func() {
			if err != nil {
				server.FailureCount++
				b, err := jsoniter.Marshal(server)
				if err == nil {
					updated = b
				}
			}
		}()
		mng, err := NewManager(ManageArgument{
			Host:     server.Host,
			Port:     strconv.Itoa(server.Port),
			Argument: server.ManageArgument,
		})
		if err != nil {
			log.Warn("NewManager(%v): %v", server.Name, err)
			return nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err = mng.Ping(ctx); err != nil {
			log.Warn("Ping: %v", err)
			return nil
		}
		server.LastSeen = time.Now()
		b, err = jsoniter.Marshal(server)
		if err != nil {
			return nil
		}
		return b
	})()
}

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

func TickUpdateBackground(bucket string, interval time.Duration, f func(b []byte, now time.Time) (updated []byte)) func() {
	type toUpdate struct {
		key []byte
		val []byte
	}
	return func() {
		tick := time.Tick(interval)
		for now := range tick {
			go func(now time.Time) {
				// mu projects the listUpdate
				var mu sync.Mutex
				var listUpdate []toUpdate
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
							if updated := f(b, now); updated != nil {
								mu.Lock()
								listUpdate = append(listUpdate, toUpdate{
									key: k,
									val: updated,
								})
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
				if len(listUpdate) == 0 {
					return
				}
				if err := db.DB().Update(func(tx *bolt.Tx) error {
					bkt, err := tx.CreateBucketIfNotExists([]byte(bucket))
					if err != nil {
						return err
					}
					for _, k := range listUpdate {
						if err := bkt.Put(k.key, k.val); err != nil {
							return err
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
