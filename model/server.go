package model

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"github.com/boltdb/bolt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/db"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/pkg/log"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/pkg/manager"
	jsoniter "github.com/json-iterator/go"
	"strconv"
	"time"
)

const (
	BucketServer = "server"
)

type ProxyProtocol string

const (
	VMessTCP    ProxyProtocol = "vmess"
	Shadowsocks               = "shadowsocks"
)

func (p ProxyProtocol) Valid() bool {
	switch p {
	case VMessTCP, Shadowsocks:
		return true
	default:
		return false
	}
}

type Server struct {
	// Every server should have a server ticket, which should be included in each API interactions
	Ticket string
	// Name is also the proxy node name
	Name string
	// Host can be either IP or domain
	Host string
	// Port is shared by management and proxy
	Port int
	// FailureCount is the count of failed ping
	FailureCount int
	// LastSeen is the time of last succeed ping
	LastSeen time.Time
	// ManageArgument is used to connect and manage the server
	ManageArgument Argument
}

func (s *Server) GetUserArgument(userTicket string) Argument {
	h := sha1.New()
	h.Write([]byte(s.Ticket))
	h.Write([]byte(userTicket))
	b := h.Sum(nil)
	return Argument{
		Protocol: Shadowsocks,
		Password: hex.EncodeToString(b)[:21],
		Method:   "chacha20-ietf-poly1305",
	}
}

type Argument struct {
	// Required
	Protocol ProxyProtocol
	// Optional
	Username string
	// Required
	Password string
	// Optional
	Method string
}

func init() {
	// remove servers that do not be seen 30 days
	go ExpireCleanBackground(BucketServer, 6*time.Hour, func(b []byte, now time.Time) (expired bool) {
		var server Server
		if err := jsoniter.Unmarshal(b, &server); err != nil {
			return false
		}
		if server.FailureCount >= 10 && time.Since(server.LastSeen).Hours() > 24*30 {
			// remove corresponding ticket
			if err := db.DB().Update(func(tx *bolt.Tx) error {
				bkt := tx.Bucket([]byte(BucketTicket))
				if bkt == nil {
					return bolt.ErrBucketNotFound
				}
				if err := bkt.Delete([]byte(server.Ticket)); err != nil {
					return err
				}
				return nil
			}); err != nil {
				log.Warn("remove expired server (%v) ticket (%v) fail: %v", server.Name, server.Ticket, err)
				// do not remove this server to avoid inconsistent data
				return false
			} else {
				return true
			}
		} else {
			return false
		}
	})

	// ping at intervals
	go TickUpdateBackground(BucketServer, 60*time.Second, func(b []byte, now time.Time) (updated []byte) {
		var server Server
		if err := jsoniter.Unmarshal(b, &server); err != nil {
			return nil
		}
		if server.FailureCount >= 10 {
			// stop the ping and wait for the register
			return nil
		}
		var err error
		defer func() {
			if err != nil {
				server.FailureCount++
				b, err := jsoniter.Marshal(server)
				if err == nil {
					updated = b
				}
			}
		}()
		mng, err := manager.NewManager(manager.ManageArgument{
			Host:     server.Host,
			Port:     strconv.Itoa(server.Port),
			Argument: server.ManageArgument,
		})
		if err != nil {
			log.Warn("NewManager(%v): %v", server.Name, err)
			return nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err = mng.Ping(ctx); err != nil {
			return nil
		}
		server.LastSeen = time.Now()
		b, err = jsoniter.Marshal(server)
		if err != nil {
			return nil
		}
		return b
	})()
}
