package service

import (
	"context"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/db"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/model"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/pkg/log"
	jsoniter "github.com/json-iterator/go"
	"strconv"
	"strings"
	"sync"
	"time"
)

func SetServerSyncNextSeenByTicket(wtx *bolt.Tx, ticket string, setTo bool) error {
	log.Info("set %v to %v for server %v", strconv.Quote("SyncNextSeen"), setTo, ticket)
	f := func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(model.BucketServer))
		var svr model.Server
		if err := jsoniter.Unmarshal(bkt.Get([]byte(ticket)), &svr); err != nil {
			return err
		}
		svr.SyncNextSeen = setTo
		b, err := jsoniter.Marshal(svr)
		if err != nil {
			return err
		}
		return bkt.Put([]byte(svr.Ticket), b)
	}
	if wtx != nil {
		return f(wtx)
	}
	return db.DB().Update(f)
}

// SyncPassagesByServer costs long time, thus tx here should be nil.
func SyncPassagesByServer(wtx *bolt.Tx, ctx context.Context, server model.Server) (err error) {
	defer func() {
		if err != nil && !server.SyncNextSeen {
			_ = SetServerSyncNextSeenByTicket(wtx, server.Ticket, true)
		}
		if err == nil && server.SyncNextSeen {
			_ = SetServerSyncNextSeenByTicket(wtx, server.Ticket, false)
		}
	}()
	passages := GetPassagesByServer(wtx, server)
	mng, err := model.NewManager(model.ManageArgument{
		Host:     server.Host,
		Port:     strconv.Itoa(server.Port),
		Argument: server.Argument,
	})
	if err != nil {
		return err
	}
	return mng.SyncPassages(ctx, passages)
}

// SyncPassagesByChatIdentifier costs long time, thus tx here should be nil.
func SyncPassagesByChatIdentifier(wtx *bolt.Tx, ctx context.Context, chatIdentifier string) (err error) {
	servers, err := GetServersByChatIdentifier(wtx, chatIdentifier)
	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	var errs []string
	var mu sync.Mutex
	for _, svr := range servers {
		passages := GetPassagesByServer(wtx, svr)
		wg.Add(1)
		go func(svr model.Server, passages []model.Passage) {
			log.Trace("SyncPassagesByChatIdentifier: chat: %v, svr: %v, passages: %v", chatIdentifier, svr, passages)
			defer wg.Done()
			defer func() {
				if err != nil && !svr.SyncNextSeen {
					_ = SetServerSyncNextSeenByTicket(wtx, svr.Ticket, true)
				}
				if err == nil && svr.SyncNextSeen {
					_ = SetServerSyncNextSeenByTicket(wtx, svr.Ticket, false)
				}
			}()
			mng, err := model.NewManager(model.ManageArgument{
				Host:     svr.Host,
				Port:     strconv.Itoa(svr.Port),
				Argument: svr.Argument,
			})
			if err != nil {
				mu.Lock()
				errs = append(errs, err.Error())
				mu.Unlock()
				return
			}
			ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
			defer cancel()
			if err = mng.SyncPassages(ctx, passages); err != nil {
				mu.Lock()
				errs = append(errs, "SyncPassages: "+err.Error())
				mu.Unlock()
				return
			}
		}(svr, passages)
	}
	wg.Wait()
	if errs != nil {
		return fmt.Errorf(strings.Join(errs, "\n"))
	}
	return nil
}
