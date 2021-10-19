package service

import (
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/db"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/model"
	jsoniter "github.com/json-iterator/go"
	"time"
)

// RegisterServer registers a server
func RegisterServer(server model.Server) (err error) {
	server.FailureCount = 0
	server.LastSeen = time.Now()
	if err := db.DB().Update(func(tx *bolt.Tx) error {
		bkt, err := tx.CreateBucketIfNotExists([]byte(model.BucketServer))
		if err != nil {
			return err
		}
		b, err := jsoniter.Marshal(&server)
		if err != nil {
			return err
		}
		return bkt.Put([]byte(server.Ticket), b)
	}); err != nil {
		return fmt.Errorf("RegisterServer: %w", err)
	}
	return nil
}
