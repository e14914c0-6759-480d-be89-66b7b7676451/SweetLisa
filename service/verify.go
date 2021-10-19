package service

import (
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/common"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/db"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/model"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/pkg/log"
	jsoniter "github.com/json-iterator/go"
	"github.com/matoous/go-nanoid/v2"
	"time"
)

// NewVerification generates a new verification and returns the verificationCode
func NewVerification(wtx *bolt.Tx, chatIdentifier string) (verificationCode string, err error) {
	if chatIdentifier == "" {
		return "", fmt.Errorf("chatIdentifier cannot be empty")
	}
	f := func(tx *bolt.Tx) error {
		bkt, err := tx.CreateBucketIfNotExists([]byte(model.BucketVerification))
		if err != nil {
			return err
		}
		for {
			id, err := gonanoid.Generate(common.Alphabet, 21)
			if err != nil {
				return err
			}
			if bkt.Get([]byte(id)) == nil {
				verificationCode = id
				break
			}
		}
		verification := model.Verification{
			Code:           verificationCode,
			ExpireAt:       time.Now().Add(1 * time.Minute),
			ChatIdentifier: chatIdentifier,
			Progress:       model.VerificationWaiting,
		}
		b, err := jsoniter.Marshal(&verification)
		if err != nil {
			return err
		}
		return bkt.Put([]byte(verificationCode), b)
	}
	if wtx != nil {
		if err = f(wtx); err != nil {
			return "", err
		}
		return verificationCode, nil
	}
	if err = db.DB().Update(f); err != nil {
		return "", err
	}
	return verificationCode, nil
}

// Verify verifies if given verificationCode and chatIdentifier can pass the verification
func Verify(wtx *bolt.Tx, verificationCode string, chatIdentifier string) error {
	f := func(tx *bolt.Tx) error {
		bkt, err := tx.CreateBucketIfNotExists([]byte(model.BucketVerification))
		if err != nil {
			return err
		}
		val := bkt.Get([]byte(verificationCode))
		// verification code was not found
		if val == nil {
			return fmt.Errorf("invalid verification code")
		}
		var verification model.Verification
		if err := jsoniter.Unmarshal(val, &verification); err != nil {
			log.Warn("%v", err)
			return fmt.Errorf("internal error")
		}
		// verification code is not for this chat
		if verification.ChatIdentifier != chatIdentifier {
			return fmt.Errorf("invalid verification code")
		}
		if common.Expired(verification.ExpireAt) {
			return model.VerificationExpiredErr
		}
		// verification has done
		if verification.Progress != model.VerificationWaiting {
			return fmt.Errorf("pass already")
		}
		verification.Progress = model.VerificationDone
		verification.ExpireAt = time.Now().Add(2 * time.Minute)
		b, err := jsoniter.Marshal(verification)
		if err != nil {
			log.Warn("%v", err)
			return fmt.Errorf("internal error")
		}

		return bkt.Put([]byte(verificationCode), b)
	}
	if wtx != nil {
		return f(wtx)
	}
	return db.DB().Update(f)
}

// Verified check if given verificationCode and chatIdentifier verification has passed
func Verified(wtx *bolt.Tx, verificationCode string, chatIdentifier string) error {
	f := func(tx *bolt.Tx) error {
		bkt, err := tx.CreateBucketIfNotExists([]byte(model.BucketVerification))
		if err != nil {
			return err
		}
		val := bkt.Get([]byte(verificationCode))
		// verification code was not found
		if val == nil {
			return fmt.Errorf("invalid verification code")
		}
		var verification model.Verification
		if err := jsoniter.Unmarshal(val, &verification); err != nil {
			log.Warn("%v", err)
			return fmt.Errorf("internal error")
		}
		// verification code is not for this chat
		if verification.ChatIdentifier != chatIdentifier {
			return fmt.Errorf("invalid verification code")
		}
		if common.Expired(verification.ExpireAt) {
			return model.VerificationExpiredErr
		}
		// verification has done
		if verification.Progress < model.VerificationDone {
			return fmt.Errorf("invalid verification code")
		}
		return nil
	}
	if wtx != nil {
		return f(wtx)
	}
	return db.DB().Update(f)
}
