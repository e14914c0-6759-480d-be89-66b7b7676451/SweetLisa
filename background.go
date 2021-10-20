package main

import (
	"context"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/common"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/model"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/pkg/log"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/service"
	jsoniter "github.com/json-iterator/go"
	"strconv"
	"sync"
	"time"
)

func shouldTicketBeRemove(server model.Server, typ model.TicketType) bool {
	if server.FailureCount < model.MaxFailureCount {
		return false
	}
	switch typ {
	case model.TicketTypeServer:
		return time.Since(server.LastSeen) > 24*35*time.Hour
	case model.TicketTypeRelay:
		return time.Since(server.LastSeen) > 24*35*time.Hour
	default:
		log.Error("shouldTicketBeRemove: unexpected ticket type: %v", typ)
		return false
	}
}

func GoBackgrounds() {
	// remove expired verifications
	go model.ExpireCleanBackground(model.BucketVerification, 10*time.Second, func(tx *bolt.Tx, b []byte, now time.Time) (expired bool) {
		var v model.Verification
		if err := jsoniter.Unmarshal(b, &v); err != nil {
			// invalid verifications are regarded as expired
			return true
		}
		return common.Expired(v.ExpireAt)
	})()

	// remove expired tickets
	go model.ExpireCleanBackground(model.BucketVerification, 5*time.Minute, func(tx *bolt.Tx, b []byte, now time.Time) (expired bool) {
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

	// remove servers/relays that have not been seen for a long time
	go model.ExpireCleanBackground(model.BucketServer, 5*time.Minute, func(tx *bolt.Tx, b []byte, now time.Time) (expired bool) {
		var server model.Server
		if err := jsoniter.Unmarshal(b, &server); err != nil {
			return false
		}
		var ticObj model.Ticket
		bkt := tx.Bucket([]byte(model.BucketTicket))
		if bkt == nil {
			return false
		}
		if err := jsoniter.Unmarshal(bkt.Get([]byte(server.Ticket)), &ticObj); err != nil {
			log.Warn("remove expired server (%v) ticket (%v) fail: %v", server.Name, server.Ticket, err)
			return false
		}
		// Relay is a server and also a client.
		// We should remove its keys immediately once it loses connection to avoid abusing.
		if ticObj.Type == model.TicketTypeRelay && now.Sub(server.LastSeen) >= 10*time.Minute {
			go func(chatIdentifier string) {
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
				defer cancel()
				if err := service.SyncPassagesByChatIdentifier(nil, ctx, chatIdentifier); err != nil {
					log.Warn("sync passages: %v: chat: %v", err, chatIdentifier)
				}
			}(ticObj.ChatIdentifier)
		}
		return now.Sub(server.LastSeen) > 24*35*time.Hour
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
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := service.Ping(ctx, server); err != nil {
			log.Info("server %v: %v", strconv.Quote(server.Name), err)
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
	for _, chatIdentifier := range identifiers {
		wg.Add(1)
		go func(chatIdentifier string) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			servers, _ := service.GetServersByChatIdentifier(nil, chatIdentifier)
			var chatWg sync.WaitGroup
			for _, server := range servers {
				serverTicket, err := service.GetValidTicketObj(nil, server.Ticket)
				if err != nil {
					log.Warn("SyncAll: cannot get ticket of server: %v", server.Name)
					continue
				}
				if serverTicket.Type == model.TicketTypeRelay {
					chatWg.Add(1)
					go func(relay model.Server, chatIdentifier string) {
						defer chatWg.Done()
						ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
						// ping test
						defer cancel()
						if err := service.Ping(ctx, relay); err != nil {
							err = fmt.Errorf("unreachable: %w", err)
							log.Warn("failed to register: %v", err)
							return
						}
						// register
						if err := service.RegisterServer(nil, relay); err != nil {
							return
						}
					}(server, chatIdentifier)
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
