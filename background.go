package main

import (
	"context"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/common"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/db"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/model"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/pkg/log"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/service"
	jsoniter "github.com/json-iterator/go"
	"strconv"
	"sync"
	"time"
)

func GoBackgrounds() {
	// remove expired verifications
	go ExpireCleanBackground(model.BucketVerification, 10*time.Second, func(tx *bolt.Tx, b []byte, now time.Time) (expired bool, chatToSync []string) {
		var v model.Verification
		if err := jsoniter.Unmarshal(b, &v); err != nil {
			// invalid verifications are regarded as expired
			return true, nil
		}
		return common.Expired(v.ExpireAt), nil
	})()

	// remove expired user tickets.
	// remove server/relay tickets that have not been seen for a long time
	go ExpireCleanBackground(model.BucketVerification, 1*time.Hour, func(tx *bolt.Tx, b []byte, now time.Time) (expired bool, chatToSync []string) {
		var ticObj model.Ticket
		err := jsoniter.Unmarshal(b, &ticObj)
		if err != nil {
			log.Warn("clean ticket: %v", err)
			return false, nil
		}
		if common.Expired(ticObj.ExpireAt) {
			return true, []string{ticObj.ChatIdentifier}
		}
		switch ticObj.Type {
		case model.TicketTypeRelay, model.TicketTypeServer:
			bkt := tx.Bucket([]byte(model.BucketServer))
			if bkt == nil {
				break
			}
			b := bkt.Get([]byte(ticObj.Ticket))
			if b == nil {
				break
			}
			var server model.Server
			if err := jsoniter.Unmarshal(b, &server); err != nil {
				break
			}
			if now.Sub(server.LastSeen) >= 35*24*time.Hour {
				log.Info("remove server ticket %v because of long time no see", server.Name)
				return true, []string{ticObj.ChatIdentifier}
			}
		}
		return false, nil
	})()

	// remove servers that have not been seen for a long time
	go ExpireCleanBackground(model.BucketServer, 10*time.Minute, func(tx *bolt.Tx, b []byte, now time.Time) (expired bool, chatToSync []string) {
		var server model.Server
		if err := jsoniter.Unmarshal(b, &server); err != nil {
			return false, nil
		}
		var ticObj model.Ticket
		bkt := tx.Bucket([]byte(model.BucketTicket))
		if bkt == nil {
			return false, nil
		}
		if err := jsoniter.Unmarshal(bkt.Get([]byte(server.Ticket)), &ticObj); err != nil {
			return false, nil
		}
		if now.Sub(server.LastSeen) >= 10*time.Minute {
			log.Info("remove server %v (type: %v) because of long time no see", server.Name, ticObj.Type)
			return true, []string{ticObj.ChatIdentifier}
		}
		return false, nil
	})()

	// ping at intervals
	go TickUpdateBackground(model.BucketServer, 1*time.Minute, func(b []byte, now time.Time) (todo func(b []byte) []byte) {
		var server model.Server
		if err := jsoniter.Unmarshal(b, &server); err != nil {
			return nil
		}
		if server.FailureCount >= model.MaxFailureCount {
			// stop the ping and wait for the proactive register
			return nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := service.Ping(ctx, server); err != nil {
			log.Info("Ping server %v: %v", strconv.Quote(server.Name), err)
			todo = func(b []byte) []byte {
				var server model.Server
				if err := jsoniter.Unmarshal(b, &server); err != nil {
					return nil
				}
				server.FailureCount++
				b, err := jsoniter.Marshal(server)
				if err != nil {
					return nil
				}
				return b
			}
		} else {
			todo = func(b []byte) []byte {
				if server.SyncNextSeen {
					_ = service.SyncPassagesByServer(context.Background(), server.Ticket)
				}
				var server model.Server
				if err := jsoniter.Unmarshal(b, &server); err != nil {
					return nil
				}
				server.FailureCount = 0
				server.LastSeen = time.Now()
				b, err := jsoniter.Marshal(server)
				if err != nil {
					return nil
				}
				return b
			}
		}
		return todo
	})()
}

func SyncAll() {
	var identifiers []string
	var wg sync.WaitGroup
	for _, ticket := range service.GetValidTickets(nil) {
		identifiers = append(identifiers, ticket.ChatIdentifier)
	}
	identifiers = common.Deduplicate(identifiers)
	// sync each chats
	for _, chatIdentifier := range identifiers {
		wg.Add(1)
		// concurrently
		go func(chatIdentifier string) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			servers, _ := service.GetServersByChatIdentifier(nil, chatIdentifier)
			var chatWg sync.WaitGroup
			// For each chat, sync servers in the chat concurrently.
			for _, server := range servers {
				serverTicket, err := service.GetValidTicketObj(nil, server.Ticket)
				if err != nil {
					log.Warn("SyncAll: cannot get ticket of server: %v", server.Name)
					continue
				}
				// For the relay, we should confirm it is reachable before registering.
				if serverTicket.Type == model.TicketTypeRelay {
					chatWg.Add(1)
					go func(ctx context.Context, relay model.Server, chatIdentifier string) {
						defer chatWg.Done()
						// ping test
						if err := service.Ping(ctx, relay); err != nil {
							err = fmt.Errorf("unreachable: %w", err)
							log.Warn("failed to register %v: %v", relay.Name, err)
							return
						}
						// register
						if err := service.RegisterServer(nil, relay); err != nil {
							return
						}
					}(ctx, server, chatIdentifier)
				}
			}
			chatWg.Wait()
			if err := service.SyncPassagesByChatIdentifier(nil, ctx, chatIdentifier); err != nil {
				log.Warn("SyncAll: %v", err)
			}
			log.Info("SyncAll for chat %v has finished", chatIdentifier)
		}(chatIdentifier)
	}
	wg.Wait()
}

func ExpireCleanBackground(bucket string, cleanInterval time.Duration, f func(tx *bolt.Tx, b []byte, now time.Time) (expired bool, chatToSync []string)) func() {
	return func() {
		tick := time.Tick(cleanInterval)
		for now := range tick {
			if err := db.DB().Update(func(tx *bolt.Tx) error {
				bkt, err := tx.CreateBucketIfNotExists([]byte(bucket))
				if err != nil {
					return err
				}
				var listClean [][]byte
				var chatToSync []string
				if err = bkt.ForEach(func(k, b []byte) error {
					expired, chat := f(tx, b, now)
					if expired {
						listClean = append(listClean, k)
					}
					chatToSync = append(chatToSync, chat...)
					return nil
				}); err != nil {
					return err
				}
				for _, k := range listClean {
					if err = bkt.Delete(k); err != nil {
						return err
					}
				}
				chatToSync = common.Deduplicate(chatToSync)
				for _, chat := range chatToSync {
					go func(chatIdentifier string) {
						ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
						defer cancel()
						if err := service.SyncPassagesByChatIdentifier(nil, ctx, chatIdentifier); err != nil {
							log.Warn("sync passages: %v: chat: %v", err, chatIdentifier)
						}
					}(chat)
				}
				return nil
			}); err != nil {
				log.Warn("Clean bucket %v: %v", bucket, err)
			}
		}
	}
}

// TickUpdateBackground will invoke f concurrently in view mode and then invoke non-nil todos in update mode.
func TickUpdateBackground(bucket string, interval time.Duration, f func(b []byte, now time.Time) (todo func(b []byte) []byte)) func() {
	return func() {
		type keyTodo struct {
			Key  []byte
			Todo func(b []byte) []byte
		}
		tick := time.Tick(interval)
		for now := range tick {
			go func(now time.Time) {
				// mu protects the keysTodo
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
