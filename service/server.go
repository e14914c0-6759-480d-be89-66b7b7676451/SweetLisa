package service

import (
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/common"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/db"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/model"
	jsoniter "github.com/json-iterator/go"
)

func GetServerByTicket(ticket string) (server model.Server, err error) {
	if err := db.DB().View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(model.BucketServer))
		if bkt == nil {
			return bolt.ErrBucketNotFound
		}
		b := bkt.Get([]byte(ticket))
		if b == nil {
			return fmt.Errorf("%w: the server may not be registered", db.ErrKeyNotFound)
		}
		return jsoniter.Unmarshal(b, &server)
	}); err != nil {
		return model.Server{}, fmt.Errorf("GetServerByTicket: %w", err)
	}
	return server, nil
}

func GetServersByChatIdentifier(chatIdentifier string) (keys []model.Server, err error) {
	var servers []model.Server
	db.DB().View(func(tx *bolt.Tx) error {
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
	})
	return servers, nil
}
