package service

import (
	"github.com/boltdb/bolt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/db"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/model"
	jsoniter "github.com/json-iterator/go"
	"time"
)

func GetKeys(chatIdentifier string, server model.Server) (keys []model.Argument) {
	db.DB().View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(model.BucketTicket))
		if bkt == nil {
			return nil
		}
		return bkt.ForEach(func(k, v []byte) error {
			var ticket model.Ticket
			if err := jsoniter.Unmarshal(v, &ticket); err != nil {
				// do not stop the iter
				return nil
			}
			if ticket.Type != model.TicketTypeUser {
				return nil
			}
			// zero means never expire
			if !ticket.ExpireAt.IsZero() && ticket.ExpireAt.Before(time.Now()) {
				return nil
			}
			keys = append(keys, server.GetUserArgument(ticket.Ticket))
			return nil
		})
	})
	return keys
}
