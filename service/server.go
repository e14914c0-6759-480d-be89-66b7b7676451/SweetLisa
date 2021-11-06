package service

import (
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/common"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/db"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/model"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/pkg/log"
	jsoniter "github.com/json-iterator/go"
	"time"
)

func GetServerByTicket(tx *bolt.Tx, ticket string) (server model.Server, err error) {
	f := func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(model.BucketServer))
		if bkt == nil {
			return bolt.ErrBucketNotFound
		}
		b := bkt.Get([]byte(ticket))
		if b == nil {
			return fmt.Errorf("%w: the server may not be registered", db.ErrKeyNotFound)
		}
		return jsoniter.Unmarshal(b, &server)
	}
	if tx != nil {
		if err = f(tx); err != nil {
			return model.Server{}, fmt.Errorf("GetServerByTicket: %w", err)
		}
		return server, nil
	}
	if err := db.DB().View(f); err != nil {
		return model.Server{}, fmt.Errorf("GetServerByTicket: %w", err)
	}
	return server, nil
}

func GetServersByChatIdentifier(tx *bolt.Tx, chatIdentifier string, includeRelay bool) (servers []model.Server, err error) {
	f := func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(model.BucketTicket))
		if bkt == nil {
			return nil
		}
		serverBkt := tx.Bucket([]byte(model.BucketServer))
		if serverBkt == nil {
			return nil
		}
		// get servers
		bkt.ForEach(func(k, v []byte) error {
			var tic model.Ticket
			if err := jsoniter.Unmarshal(v, &tic); err != nil {
				return nil
			}
			if tic.ChatIdentifier != chatIdentifier ||
				common.Expired(tic.ExpireAt) {
				return nil
			}
			switch tic.Type {
			case model.TicketTypeServer, model.TicketTypeRelay:
				if tic.Type == model.TicketTypeRelay && !includeRelay {
					return nil
				}
			default:
				return nil
			}
			var svr model.Server
			bServer := serverBkt.Get([]byte(tic.Ticket))
			if err := jsoniter.Unmarshal(bServer, &svr); err != nil {
				return nil
			}
			servers = append(servers, svr)
			return nil
		})
		return nil
	}
	if tx != nil {
		f(tx)
		return servers, nil
	}
	db.DB().View(f)
	return servers, nil
}

// RegisterServer save the server in db
func RegisterServer(wtx *bolt.Tx, server model.Server) (err error) {
	f := func(tx *bolt.Tx) error {
		bkt, err := tx.CreateBucketIfNotExists([]byte(model.BucketServer))
		if err != nil {
			return err
		}
		// register a new server
		var old model.Server
		if bOld := bkt.Get([]byte(server.Ticket)); bOld == nil {
			if err := AddFeedServer(tx, server, ServerActionLaunch); err != nil {
				return err
			}
		} else {
			if err := jsoniter.Unmarshal(bOld, &old); err != nil {
				return err
			}
		}
		if old.FailureCount >= model.MaxFailureCount {
			log.Info("server %v reconnected", server.Name)
			_ = AddFeedServer(wtx, server, ServerActionReconnect)
		}
		old.BandwidthLimit.Update(server.BandwidthLimit)
		server.BandwidthLimit = old.BandwidthLimit

		server.FailureCount = 0
		server.LastSeen = time.Now()
		server.SyncNextSeen = false

		b, err := jsoniter.Marshal(&server)
		if err != nil {
			return err
		}
		return bkt.Put([]byte(server.Ticket), b)
	}
	if wtx != nil {
		if err := f(wtx); err != nil {
			return fmt.Errorf("RegisterServer: %w", err)
		}
		return nil
	}

	if err := db.DB().Update(f); err != nil {
		return fmt.Errorf("RegisterServer: %w", err)
	}
	return nil
}
