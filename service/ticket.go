package service

import (
	"encoding/base64"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/cryptoballot/rsablind"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/db"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/model"
	jsoniter "github.com/json-iterator/go"
	"time"
)

// BlindSign signs a blinded ticket using the chat unique private key.
// blindedTicket should be a string encoded by URL friendly base64.
func BlindSign(blindedTicket string, chatIdentifier string) (sig []byte, err error) {
	key, err := GetChatBlindKey(chatIdentifier)
	if err != nil {
		return nil, err
	}
	blinded, err := base64.URLEncoding.DecodeString(blindedTicket)
	if err != nil {
		return nil, err
	}
	sig, err = rsablind.BlindSign(key, blinded)
	if err != nil {
		return nil, err
	}
	return sig, nil
}

// SaveSig saves the given sig to the database and sets the expiration time to the next month
func SaveSig(sigBytes []byte) (sig model.Sig, err error) {
	sig = model.Sig{
		Sig:    base64.URLEncoding.EncodeToString(sigBytes),
		Expire: time.Now().AddDate(0, 1, 0),
	}
	return sig, db.DB().Update(func(tx *bolt.Tx) error {
		bkt, err := tx.CreateBucketIfNotExists([]byte(model.BucketSig))
		if err != nil {
			return err
		}
		b, err := jsoniter.Marshal(&sig)
		if err != nil {
			return err
		}
		return bkt.Put(sigBytes, b)
	})
}

func IsTicketValid(ticket string, sig string, chatIdentifier string) error {
	key, err := GetChatBlindKey(chatIdentifier)
	if err != nil {
		return err
	}
	sigBytes, err := base64.URLEncoding.DecodeString(sig)
	if err != nil {
		return err
	}
	if err = rsablind.VerifyBlindSignature(&key.PublicKey, []byte(ticket), sigBytes); err != nil {
		return err
	}
	return db.DB().View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(model.BucketSig))
		if bkt == nil {
			return fmt.Errorf("bucket %v does not exist", model.BucketSig)
		}
		b := bkt.Get(sigBytes)
		if b == nil {
			return fmt.Errorf("invalid sig")
		}
		var sig model.Sig
		if err := jsoniter.Unmarshal(b, &sig); err != nil {
			return err
		}
		if time.Now().After(sig.Expire) {
			return fmt.Errorf("invalid sig")
		}
		return nil
	})
}
