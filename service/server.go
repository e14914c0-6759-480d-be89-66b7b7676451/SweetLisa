package service

import (
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/db"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/model"
	jsoniter "github.com/json-iterator/go"
)

func GetServerByTicket(ticket string) (server model.Server, err error) {
	if err := db.DB().View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(model.BucketTicket))
		if bkt == nil {
			return bolt.ErrBucketNotFound
		}
		b := bkt.Get([]byte(ticket))
		if b == nil {
			return db.ErrKeyNotFound
		}
		return jsoniter.Unmarshal(b, &server)
	}); err != nil {
		return model.Server{}, fmt.Errorf("GetServerByTicket: %w", err)
	}
	return server, nil
}
