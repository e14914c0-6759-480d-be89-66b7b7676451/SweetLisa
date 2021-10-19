package main

import (
	"context"
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

func shouldBeRemove(server model.Server, typ model.TicketType) bool {
	if server.FailureCount < model.MaxFailureCount {
		return false
	}
	switch typ {
	case model.TicketTypeServer:
		return time.Since(server.LastSeen) > 24*35*time.Hour
	case model.TicketTypeRelay:
		return time.Since(server.LastSeen) > 10*time.Minute
	default:
		log.Error("shouldBeRemove: unexpected ticket type: %v", typ)
		return false
	}
}

func GoBackgrounds() {
	// remove expired verifications
	go model.ExpireCleanBackground(model.BucketVerification, 10*time.Second, func(b []byte, now time.Time) (expired bool) {
		var v model.Verification
		if err := jsoniter.Unmarshal(b, &v); err != nil {
			// invalid verifications are regarded as expired
			return true
		}
		return common.Expired(v.ExpireAt)
	})()

	// remove expired tickets
	go model.ExpireCleanBackground(model.BucketVerification, 5*time.Minute, func(b []byte, now time.Time) (expired bool) {
		var ticket model.Ticket
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

	// remove relays that do not be seen 10 minutes
	go model.ExpireCleanBackground(model.BucketServer, 5*time.Minute, func(b []byte, now time.Time) (expired bool) {
		var server model.Server
		if err := jsoniter.Unmarshal(b, &server); err != nil {
			return false
		}
		var ticObj model.Ticket
		if err := db.DB().Update(func(tx *bolt.Tx) error {
			bkt := tx.Bucket([]byte(model.BucketTicket))
			if bkt == nil {
				return bolt.ErrBucketNotFound
			}
			if err := jsoniter.Unmarshal(bkt.Get([]byte(server.Ticket)), &ticObj); err != nil {
				return err
			}
			// should corresponding ticket be remove?
			if expired = shouldBeRemove(server, ticObj.Type); expired {
				if err := bkt.Delete([]byte(server.Ticket)); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			log.Warn("remove expired server (%v) ticket (%v) fail: %v", server.Name, server.Ticket, err)
			// do not remove this server to avoid inconsistent data
			return false
		}
		// Relay is a server and also a client.
		// We should remove its keys immediately once it loses connection to avoid abusing.
		if expired && ticObj.Type == model.TicketTypeRelay {
			go func(chatIdentifier string) {
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
				defer cancel()
				if err := service.SyncKeysByChatIdentifier(nil, ctx, chatIdentifier); err != nil {
					log.Warn("sync keys: %v: chat: %v", err, chatIdentifier)
				}
			}(ticObj.ChatIdentifier)
		}
		return true
	})()

	// ping at intervals
	go model.TickUpdateBackground(model.BucketServer, 1*time.Minute, func(b []byte, now time.Time) (todo func(b []byte) []byte) {
		var server model.Server
		if err := jsoniter.Unmarshal(b, &server); err != nil {
			return nil
		}
		if server.FailureCount >= model.MaxFailureCount {
			// stop the ping and wait for the register
			return nil
		}
		var err error
		defer func() {
			if err != nil {
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
				if server.SyncNextSeen {
					todo = func(b []byte) []byte {
						go func() {
							ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
							defer cancel()
							// Run SyncKeysByServer in a new coroutine to avoid nested transactions and deadlock
							_ = service.SyncKeysByServer(nil, ctx, server)
						}()
						return nil
					}
				}
			}
		}()
		mng, err := model.NewManager(model.ManageArgument{
			Host:     server.Host,
			Port:     strconv.Itoa(server.Port),
			Argument: server.Argument,
		})
		if err != nil {
			log.Warn("NewManager(%v): %v", server.Name, err)
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err = mng.Ping(ctx); err != nil {
			log.Warn("Ping: %v", err)
			return
		}
		return
	})()
}

func SyncAll() {
	var identifiers []string
	var wg sync.WaitGroup
	for _, ticket := range service.GetValidTickets(nil) {
		identifiers = append(identifiers, ticket.ChatIdentifier)
	}
	identifiers = common.Deduplicate(identifiers)
	for _, chatIdentifier := range identifiers {
		wg.Add(1)
		go func(chatIdentifier string) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := service.SyncKeysByChatIdentifier(nil, ctx, chatIdentifier); err != nil {
				log.Warn("SyncAll: %v", err)
			}
		}(chatIdentifier)
	}
	wg.Wait()
}
