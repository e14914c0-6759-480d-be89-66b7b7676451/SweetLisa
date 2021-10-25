package model

import (
	"github.com/gorilla/feeds"
)

const (
	BucketFeed = "feed"
)

type ChatFeed struct {
	ChatIdentifier string
	Feeds          []*feeds.Item
}
