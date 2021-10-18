package service

import (
	"context"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/common"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/db"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/model"
	jsoniter "github.com/json-iterator/go"
	"strconv"
	"strings"
	"sync"
	"time"
)

func SyncKeysByServer(ctx context.Context, server model.Server) (err error) {
	keys := GetKeysByServer(server)
	mng, err := model.NewManager(model.ManageArgument{
		Host:     server.Host,
		Port:     strconv.Itoa(server.Port),
		Argument: server.ManageArgument,
	})
	return mng.SyncKeys(ctx, keys)
}

func SyncKeysByChatIdentifier(ctx context.Context, chatIdentifier string) (err error) {
	var servers []model.Server
	db.DB().View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(model.BucketTicket))
		if bkt == nil {
			return nil
		}
		serverBkt := tx.Bucket([]byte(model.BucketServer))
		if serverBkt == nil {
			return nil
		}
		// get servers
		bkt.ForEach(func(k, v []byte) error {
			var tic model.Ticket
			if err := jsoniter.Unmarshal(v, &tic); err != nil {
				return nil
			}
			if tic.ChatIdentifier != chatIdentifier ||
				common.Expired(tic.ExpireAt) ||
				tic.Type != model.TicketTypeServer {
				return nil
			}
			var svr model.Server
			bServer := serverBkt.Get([]byte(tic.Ticket))
			if err := jsoniter.Unmarshal(bServer, &svr); err != nil {
				return nil
			}
			servers = append(servers, svr)
			return nil
		})
		return nil
	})
	var wg sync.WaitGroup
	var errs []string
	var mu sync.Mutex
	for _, svr := range servers {
		keys := GetKeysByServer(svr)
		wg.Add(1)
		go func(svr model.Server, keys []model.Argument) {
			defer wg.Done()
			mng, err := model.NewManager(model.ManageArgument{
				Host:     svr.Host,
				Port:     strconv.Itoa(svr.Port),
				Argument: svr.ManageArgument,
			})
			if err != nil {
				mu.Lock()
				errs = append(errs, err.Error())
				mu.Unlock()
				return
			}
			ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
			defer cancel()
			if err = mng.SyncKeys(ctx, keys); err != nil {
				mu.Lock()
				errs = append(errs, err.Error())
				mu.Unlock()
				return
			}
		}(svr, keys)
	}
	wg.Wait()
	if errs != nil {
		return fmt.Errorf(strings.Join(errs, "\n"))
	}
	return nil
}
