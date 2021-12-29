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

var ErrInvalidTicket = fmt.Errorf("invalid ticket")

// SaveTicket saves the given ticket to the database and sets the expiration time to the next month
func SaveTicket(wtx *bolt.Tx, ticket string, typ model.TicketType, chatIdentifier string) (tic model.Ticket, err error) {
	tic = model.Ticket{
		Ticket:         ticket,
		ChatIdentifier: chatIdentifier,
		Type:           typ,
	}
	// server ticket never expire
	switch typ {
	case model.TicketTypeUser:
		tic.ExpireAt = time.Now().AddDate(0, 1, 0)
	case model.TicketTypeServer, model.TicketTypeRelay:
		tic.ExpireAt = time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)
	default:
		err = fmt.Errorf("unexpected ticket type: %v", tic.Type)
		log.Error("%v", err)
		return model.Ticket{}, err
	}
	f := func(tx *bolt.Tx) error {
		bkt, err := tx.CreateBucketIfNotExists([]byte(model.BucketTicket))
		if err != nil {
			return err
		}
		b, err := jsoniter.Marshal(&tic)
		if err != nil {
			return err
		}
		return bkt.Put([]byte(ticket), b)
	}
	if wtx != nil {
		return tic, f(wtx)
	}
	return tic, db.DB().Update(f)
}

func GetTicketObj(tx *bolt.Tx, ticket string) (tic model.Ticket, err error) {
	f := func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(model.BucketTicket))
		if bkt == nil {
			return fmt.Errorf("bucket %v does not exist", model.BucketTicket)
		}
		b := bkt.Get([]byte(ticket))
		if b == nil {
			return ErrInvalidTicket
		}
		var t model.Ticket
		if err := jsoniter.Unmarshal(b, &t); err != nil {
			return err
		}
		tic = t
		return nil
	}
	if tx != nil {
		if err = f(tx); err != nil {
			return model.Ticket{}, err
		}
		return tic, nil
	}
	if err = db.DB().View(f); err != nil {
		return model.Ticket{}, err
	}
	return tic, nil
}

// GetValidTicketObj returns ticket object if given ticket is valid
func GetValidTicketObj(tx *bolt.Tx, ticket string) (tic model.Ticket, err error) {
	defer func() {
		// zero means never expire
		if err == nil && common.Expired(tic.ExpireAt) {
			err = fmt.Errorf("%w: expired", ErrInvalidTicket)
		}
	}()
	if tx != nil {
		if tic, err = GetTicketObj(tx, ticket); err != nil {
			return model.Ticket{}, err
		}
		return tic, nil
	}
	if err = db.DB().View(func(tx *bolt.Tx) error {
		tic, err = GetTicketObj(tx, ticket)
		return err
	}); err != nil {
		return model.Ticket{}, err
	}
	return tic, nil
}

func GetValidTickets(tx *bolt.Tx) (tickets []model.Ticket) {
	f := func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(model.BucketTicket))
		if bkt == nil {
			return fmt.Errorf("bucket %v does not exist", model.BucketTicket)
		}
		return bkt.ForEach(func(k, v []byte) error {
			var t model.Ticket
			if err := jsoniter.Unmarshal(v, &t); err != nil {
				return nil
			}
			if common.Expired(t.ExpireAt) {
				return nil
			}
			tickets = append(tickets, t)
			return nil
		})
	}
	if tx != nil {
		_ = f(tx)
		return tickets
	}
	_ = db.DB().View(f)
	return tickets
}

func RevokeTicket(wtx *bolt.Tx, ticket string, chatIdentifier string) (err error) {
	f := func(tx *bolt.Tx) error {
		ticObj, err := GetValidTicketObj(tx, ticket)
		if err != nil {
			return err
		}
		if ticObj.ChatIdentifier != chatIdentifier {
			return ErrInvalidTicket
		}
		switch ticObj.Type {
		case model.TicketTypeServer, model.TicketTypeRelay:
			svrBkt := tx.Bucket([]byte(model.BucketServer))
			if svrBkt != nil {
				// some server/relay type tickets have not yet registered.
				// so ignore the error.
				_ = svrBkt.Delete([]byte(ticket))
			}
		default:
		}
		ticBkt, err := tx.CreateBucketIfNotExists([]byte(model.BucketTicket))
		if err != nil {
			return err
		}
		return ticBkt.Delete([]byte(ticket))
	}
	if wtx != nil {
		return f(wtx)
	}
	return db.DB().Update(f)
}
