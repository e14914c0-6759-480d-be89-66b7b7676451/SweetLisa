package service

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"github.com/boltdb/bolt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/db"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/model"
)

// GetChatBlindKey returns the chat unique RSA private key for blind sign from database
func GetChatBlindKey(chatIdentifier string) (key *rsa.PrivateKey, err error) {
	if err = db.DB().Update(func(tx *bolt.Tx) error {
		bkt, err := tx.CreateBucketIfNotExists([]byte(model.BucketChatBlindKey))
		if err != nil {
			return err
		}
		b := bkt.Get([]byte(chatIdentifier))
		if b != nil {
			k, e := x509.ParsePKCS1PrivateKey(b)
			if e != nil {
				return e
			}
			key = k
			return nil
		} else {
			// no blind key found. generate one
			k, e := rsa.GenerateKey(rand.Reader, model.BlindKeySize)
			if e != nil {
				return e
			}
			key = k
			return bkt.Put([]byte(chatIdentifier), pem.EncodeToMemory(&pem.Block{
				Type:  "RSA PRIVATE KEY",
				Bytes: x509.MarshalPKCS1PrivateKey(key),
			}))
		}
	}); err != nil {
		return nil, err
	}
	return key, nil
}
