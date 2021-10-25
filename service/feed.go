package service

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/config"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/db"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/model"
	"github.com/gorilla/feeds"
	"net/url"
	"path"
	"time"
)

type ServerAction string

const (
	ServerActionLaunch  ServerAction = "Launch"
	ServerActionOffline              = "Offline"
)

func GetChatFeedRSS(tx *bolt.Tx, chatIdentifier string) (string, error) {
	var feedItems []*feeds.Item
	f := func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(model.BucketFeed))
		if bkt == nil {
			return nil
		}
		b := bkt.Get([]byte(chatIdentifier))
		var chatFeedObj model.ChatFeed
		if b == nil {
			return nil
		}
		if err := gob.NewDecoder(bytes.NewReader(b)).Decode(&chatFeedObj); err != nil {
			return err
		}
		feedItems = chatFeedObj.Feeds
		return nil
	}
	if tx != nil {
		if err := f(tx); err != nil {
			return "", fmt.Errorf("GetFeed: %w", err)
		}
	} else {
		if err := db.DB().View(f); err != nil {
			return "", fmt.Errorf("GetFeed: %w", err)
		}
	}
	now := time.Now()
	chatLink := url.URL{
		Scheme: "https",
		Host:   config.GetConfig().Host,
		Path:   path.Join("chat", chatIdentifier),
	}
	feed := feeds.Feed{
		Title:       "Republic of Developers Airline (aka RDA)",
		Link:        &feeds.Link{Href: chatLink.String()},
		Description: chatIdentifier,
		Author:      &feeds.Author{Name: "Sweet Lisa", Email: "@SweetLisa_bot"},
		Created:     now,
		Items:       feedItems,
	}
	return feed.ToRss()
}

func AddFeed(wtx *bolt.Tx, chatIdentifier string, item feeds.Item) error {
	f := func(tx *bolt.Tx) error {
		bkt, err := tx.CreateBucketIfNotExists([]byte(model.BucketFeed))
		if err != nil {
			return err
		}
		b := bkt.Get([]byte(chatIdentifier))
		var chatFeedObj model.ChatFeed
		if b != nil {
			if err := gob.NewDecoder(bytes.NewReader(b)).Decode(&chatFeedObj); err != nil {
				return err
			}
		} else {
			chatFeedObj.ChatIdentifier = chatIdentifier
		}
		chatFeedObj.Feeds = append(chatFeedObj.Feeds, &item)
		var buf bytes.Buffer
		err = gob.NewEncoder(&buf).Encode(&chatFeedObj)
		if err != nil {
			return err
		}
		return bkt.Put([]byte(chatIdentifier), buf.Bytes())
	}
	if wtx != nil {
		if err := f(wtx); err != nil {
			return fmt.Errorf("AddFeed: %w", err)
		}
		return nil
	}

	if err := db.DB().Update(f); err != nil {
		return fmt.Errorf("AddFeed: %w", err)
	}
	return nil
}

func AddFeedServer(wtx *bolt.Tx, server model.Server, action ServerAction) (err error) {
	tic, err := GetValidTicketObj(wtx, server.Ticket)
	if err != nil {
		return err
	}
	var typ string
	switch tic.Type {
	case model.TicketTypeServer:
		typ = "Endpoint Server"
	case model.TicketTypeRelay:
		typ = "Relay Server"
	}
	return AddFeed(wtx, tic.ChatIdentifier, feeds.Item{
		Title: fmt.Sprintf("Servers Changed"),
		Link: &feeds.Link{
			Href: "",
		},
		Description: fmt.Sprintf("%v (%v): %v", action, typ, server.Name),
		Created:     time.Now(),
	})
}
