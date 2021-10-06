package service

import (
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/db"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/model"
	jsoniter "github.com/json-iterator/go"
	"time"
)

// SaveTicket saves the given ticket to the database and sets the expiration time to the next month
func SaveTicket(ticket string, typ model.TicketType, chatIdentifier string) (tic model.Ticket, err error) {
	tic = model.Ticket{
		Ticket:         ticket,
		ChatIdentifier: chatIdentifier,
		Type:           typ,
	}
	if typ == model.TicketTypeUser {
		tic.ExpireAt = time.Now().AddDate(0, 1, 0)
	}
	return tic, db.DB().Update(func(tx *bolt.Tx) error {
		bkt, err := tx.CreateBucketIfNotExists([]byte(model.BucketTicket))
		if err != nil {
			return err
		}
		b, err := jsoniter.Marshal(&tic)
		if err != nil {
			return err
		}
		return bkt.Put([]byte(ticket), b)
	})
}

// GetValidTicketObj returns ticket object if given ticket is valid
func GetValidTicketObj(ticket string) (tic model.Ticket, err error) {
	err = db.DB().View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(model.BucketTicket))
		if bkt == nil {
			return fmt.Errorf("bucket %v does not exist", model.BucketTicket)
		}
		b := bkt.Get([]byte(ticket))
		if b == nil {
			return fmt.Errorf("invalid ticket")
		}
		var t model.Ticket
		if err := jsoniter.Unmarshal(b, &t); err != nil {
			return err
		}
		if time.Now().After(t.ExpireAt) {
			return fmt.Errorf("invalid ticket: expired")
		}
		tic = t
		return nil
	})
	if err != nil {
		return model.Ticket{}, err
	}
	return tic, nil
}
