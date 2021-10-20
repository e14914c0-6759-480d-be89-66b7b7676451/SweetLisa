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

const ServerSyncBoxCleanTimeout = 6 * time.Hour

type ServerSyncBox struct {
	waitingSync chan struct{}
	box         map[string]chan context.Context
	lastSync    map[string]time.Time
	syncCancel  map[string]func()
	mu          sync.Mutex
	closed      chan struct{}
}

func NewServerSyncBox() *ServerSyncBox {
	return &ServerSyncBox{
		waitingSync: make(chan struct{}, 1),
		box:         make(map[string]chan context.Context),
		lastSync:    make(map[string]time.Time),
		syncCancel:  make(map[string]func()),
	}
}

func (b *ServerSyncBox) ReqSync(ctx context.Context, serverTicket string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if cancel, ok := b.syncCancel[serverTicket]; ok {
		cancel()
	}

	box, ok := b.box[serverTicket]
	if !ok {
		b.box[serverTicket] = make(chan context.Context, 1)
	}
	select {
	case box <- ctx:
	default:
	}

	select {
	case b.waitingSync <- struct{}{}:
	default:
	}
}

func (b *ServerSyncBox) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	select {
	case <-b.closed:
		close(b.closed)
	default:
	}
	return nil
}

func (b *ServerSyncBox) CleanBackground() {
	for {
		select {
		case <-b.closed:
			return
		default:
			b.mu.Lock()
			var toRemove []string
			for ticket := range b.lastSync {
				if time.Since(b.lastSync[ticket]) > ServerSyncBoxCleanTimeout {
					toRemove = append(toRemove, ticket)
				}
			}
			for _, ticket := range toRemove {
				delete(b.lastSync, ticket)
				if _, ok := b.box[ticket]; ok {
					delete(b.box, ticket)
				}
			}
			b.mu.Unlock()
		}
	}
}

func (b *ServerSyncBox) SyncBackground() {
	var wg sync.WaitGroup
	for range b.waitingSync {
		b.mu.Lock()
		for ticket, ch := range b.box {
			select {
			case ctx := <-ch:
				ctx, cancel := context.WithCancel(ctx)
				b.syncCancel[ticket] = cancel
				b.lastSync[ticket] = time.Now()
				wg.Add(1)
				go func(ctx context.Context, cancel func(), ticket string) {
					defer func() {
						select {
						case <-ctx.Done():
							// cancel() was called and a new cancel will overwrite the old one.
							// So do nothing here.
						default:
							cancel()
							b.mu.Lock()
							delete(b.syncCancel, ticket)
							b.mu.Unlock()
						}
						wg.Done()
					}()
					svr, err := GetServerByTicket(nil, ticket)
					if err != nil {
						log.Info("SyncBackground: GetServerByTicket: %v", err)
						return
					}
					mng, err := model.NewManager(model.ManageArgument{
						Host:     svr.Host,
						Port:     strconv.Itoa(svr.Port),
						Argument: svr.Argument,
					})
					if err != nil {
						log.Info("SyncBackground: %v: %v", svr.Name, err)
						return
					}
					defer func() {
						if err != nil {
							log.Info("Retry the sync after seeing the server %v next time", svr.Name)
						}
						_ = setSyncNextSeen(ticket, err != nil)
					}()
					subCtx, subCancel := context.WithTimeout(ctx, 15*time.Second)
					defer subCancel()
					passages := GetPassagesByServer(nil, svr.Ticket)
					if err = mng.SyncPassages(subCtx, passages); err != nil {
						log.Info("SyncBackground: %v: %v", svr.Name, err)
						return
					}
				}(ctx, cancel, ticket)
			default:
				// no sync request for this ticket
			}
		}
		b.mu.Unlock()
		wg.Wait()
	}
}

func setSyncNextSeen(ticket string, syncNextSeen bool) error {
	return db.DB().Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(model.BucketServer))
		if bkt == nil {
			return bolt.ErrBucketNotFound
		}
		b := bkt.Get([]byte(ticket))
		if b == nil {
			return db.ErrKeyNotFound
		}
		var server model.Server
		if err := jsoniter.Unmarshal(b, &server); err != nil {
			return err
		}
		server.SyncNextSeen = syncNextSeen
		b, err := jsoniter.Marshal(server)
		if err != nil {
			return err
		}
		return bkt.Put([]byte(ticket), b)
	})
}

var DefaultServerSyncBox = NewServerSyncBox()

func init() {
	go DefaultServerSyncBox.SyncBackground()
}

func SyncPassagesByServer(ctx context.Context, serverTicket string) (err error) {
	DefaultServerSyncBox.ReqSync(ctx, serverTicket)
	return nil
}

// SyncPassagesByChatIdentifier costs long time, thus tx here should be nil.
func SyncPassagesByChatIdentifier(wtx *bolt.Tx, ctx context.Context, chatIdentifier string) (err error) {
	servers, err := GetServersByChatIdentifier(wtx, chatIdentifier)
	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	var errs []string
	for _, svr := range servers {
		DefaultServerSyncBox.ReqSync(ctx, svr.Ticket)
	}
	wg.Wait()
	if errs != nil {
		return fmt.Errorf(strings.Join(errs, "\n"))
	}
	return nil
}

func Ping(ctx context.Context, server model.Server) error {
	mng, err := model.NewManager(model.ManageArgument{
		Host:     server.Host,
		Port:     strconv.Itoa(server.Port),
		Argument: server.Argument,
	})
	if err != nil {
		return fmt.Errorf("NewManager(%v): %w", server.Name, err)
	}
	if err = mng.Ping(ctx); err != nil {
		return fmt.Errorf("Ping: %w", err)
	}
	return nil
}
