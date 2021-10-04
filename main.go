package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"github.com/cryptoballot/fdh"
	"github.com/cryptoballot/rsablind"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/pkg/log"
)

func main() {
	//go func() {
	//	_, err := bot.New(config.GetConfig().BotToken, nil)
	//	if err != nil {
	//		log.Fatal("Bot: %v", err)
	//	}
	//}()
	message := []byte("ATTACKATDAWN")

	keysize := 2048
	hashize := 1536

	// We do a SHA256 full-domain-hash expanded to 1536 bits (3/4 the key size)
	hashed := fdh.Sum(crypto.SHA256, hashize, message)

	// Generate a key
	key, _ := rsa.GenerateKey(rand.Reader, keysize)

	// Blind the hashed message
	blinded, unblinder, err := rsablind.Blind(&key.PublicKey, hashed)
	if err != nil {
		panic(err)
	}
	log.Warn("%v", blinded)

	// Blind sign the blinded message
	sig, err := rsablind.BlindSign(key, blinded)
	if err != nil {
		panic(err)
	}

	// Unblind the signature
	unblindedSig := rsablind.Unblind(&key.PublicKey, sig, unblinder)

	// Verify the original hashed message against the unblinded signature
	if err := rsablind.VerifyBlindSignature(&key.PublicKey, hashed, unblindedSig); err != nil {
		panic("failed to verify signature")
	} else {
		fmt.Println("ALL IS WELL")
	}
}
