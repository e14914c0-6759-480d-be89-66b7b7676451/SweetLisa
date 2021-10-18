package service

import (
	"github.com/boltdb/bolt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/common"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/db"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/model"
	jsoniter "github.com/json-iterator/go"
)

func GetKeysByServer(server model.Server) (keys []model.Argument) {
	db.DB().View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(model.BucketTicket))
		if bkt == nil {
			return nil
		}
		// get server chatIdentifier
		bServerTicket := bkt.Get([]byte(server.Ticket))
		var serverTicket model.Ticket
		if err := jsoniter.Unmarshal(bServerTicket, &serverTicket); err != nil {
			return err
		}
		chatIdentifier := serverTicket.ChatIdentifier
		// generate all user keys in this chat
		return bkt.ForEach(func(k, v []byte) error {
			var ticket model.Ticket
			if err := jsoniter.Unmarshal(v, &ticket); err != nil {
				// do not stop the iter
				return nil
			}
			if ticket.ChatIdentifier != chatIdentifier ||
				ticket.Type != model.TicketTypeUser {
				return nil
			}
			if common.Expired(ticket.ExpireAt) {
				return nil
			}
			keys = append(keys, server.GetUserArgument(ticket.Ticket))
			return nil
		})
	})
	return keys
}
