package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/common"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/config"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/db"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/manager"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/model"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/pkg/ipip"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/pkg/log"
	jsoniter "github.com/json-iterator/go"
	"golang.org/x/net/proxy"
	"net"
	"net/url"
	"strconv"
	"sync"
	"time"
)

const ServerSyncBoxCleanTimeout = 6 * time.Hour

var CNProxyNotSetErr = fmt.Errorf("cnproxy is not set")

type ServerSyncBox struct {
	waitingSync chan struct{}
	box         map[string]chan struct{}
	lastSync    map[string]time.Time
	syncCancel  map[string]func()
	mu          sync.Mutex
	closed      chan struct{}
}

func NewServerSyncBox() *ServerSyncBox {
	return &ServerSyncBox{
		waitingSync: make(chan struct{}, 1),
		box:         make(map[string]chan struct{}),
		lastSync:    make(map[string]time.Time),
		syncCancel:  make(map[string]func()),
	}
}

func (b *ServerSyncBox) ReqSync(serverTicket string) {
	log.Trace("ReqSync: tic: %v", serverTicket)
	b.mu.Lock()
	defer b.mu.Unlock()
	if cancel, ok := b.syncCancel[serverTicket]; ok {
		cancel()
		delete(b.syncCancel, serverTicket)
	}

	box, ok := b.box[serverTicket]
	if !ok {
		b.box[serverTicket] = make(chan struct{}, 1)
		box = b.box[serverTicket]
	}
	select {
	case box <- struct{}{}:
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
		log.Trace("Sync Scan")
		for ticket, ch := range b.box {
			select {
			case <-ch:
				ctx, cancel := context.WithCancel(context.Background())
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
					log.Trace("Sync: tic: %v: %v", ticket, svr.Name)
					mng, err := manager.NewManager(ChooseDialer(svr), manager.ManageArgument{
						Host:     model.GetFirstHost(svr.Hosts),
						Port:     strconv.Itoa(svr.Port),
						Argument: svr.Argument,
					})
					if err != nil {
						log.Info("SyncBackground: %v: %v", svr.Name, err)
						return
					}
					defer func() {
						failed := err != nil && !common.IsCanceled(err)
						if failed {
							log.Info("Retry the sync after seeing the server %v next time: %v", svr.Name, err.Error())
						}
						_ = setSyncNextSeen(ticket, failed)
					}()
					subCtx, subCancel := context.WithTimeout(ctx, 15*time.Second)
					defer subCancel()
					passages := GetPassagesByServer(nil, svr.Ticket)
					if err = mng.SyncPassages(subCtx, passages); err != nil {
						switch {
						case common.IsCanceled(err):
							// pass
							log.Trace("SyncBackground: cancel: %v", err)
						default:
							log.Info("SyncBackground (%v): %v", svr.Name, err)
						}
						return
					}
				}(ctx, cancel, ticket)
			default:
				// no sync request for this ticket
			}
		}
		b.mu.Unlock()
		wg.Wait()
		time.Sleep(5 * time.Second)
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

func ReqSyncPassagesByServer(tx *bolt.Tx, serverTicket string, onlyItSelf bool) (err error) {
	ticketsToSync := []string{serverTicket}
	tic, err := GetValidTicketObj(tx, serverTicket)
	if err != nil {
		return err
	}
	servers, err := GetServersByChatIdentifier(tx, tic.ChatIdentifier, true)
	if err != nil {
		return err
	}
	if !onlyItSelf {
		for _, svr := range servers {
			t, err := GetValidTicketObj(tx, svr.Ticket)
			if err != nil {
				continue
			}
			if tic.Type == model.TicketTypeServer && t.Type == model.TicketTypeRelay ||
				tic.Type == model.TicketTypeRelay && t.Type == model.TicketTypeServer {
				if svr.SyncNextSeen {
					continue
				}
				ticketsToSync = append(ticketsToSync, t.Ticket)
			}
		}
	}
	for _, tic := range ticketsToSync {
		DefaultServerSyncBox.ReqSync(tic)
	}
	return nil
}

// ReqSyncPassagesByChatIdentifier costs long time, thus tx here should be nil.
func ReqSyncPassagesByChatIdentifier(tx *bolt.Tx, chatIdentifier string, includeRelay bool) (err error) {
	servers, err := GetServersByChatIdentifier(tx, chatIdentifier, includeRelay)
	if err != nil {
		return err
	}
	for _, svr := range servers {
		if svr.SyncNextSeen {
			continue
		}
		DefaultServerSyncBox.ReqSync(svr.Ticket)
	}
	return nil
}

// ChooseDialer choose CNProxy dialer for servers in China, and net.Dialer for others
func ChooseDialer(server model.Server) manager.Dialer {
	cnDialer, err := GetCNProxyDialer()
	if err != nil {
		if !errors.Is(err, CNProxyNotSetErr) {
			log.Warn("ChooseDialer: %v", err)
		}
		return &net.Dialer{}
	}
	ip := model.GetFirstHost(server.Hosts)
	if net.ParseIP(ip) == nil {
		ips, err := net.LookupHost(ip)
		if err != nil {
			return &net.Dialer{}
		}
		ip = ips[0]
	}
	if !ipip.IsChinaIPLookupTable(ip) {
		return &net.Dialer{}
	}
	return &manager.DialerConverter{Dialer: cnDialer}
}

func GetCNProxyDialer() (proxy.Dialer, error) {
	cnProxy := config.GetConfig().CNProxy
	if cnProxy == "" {
		return nil, CNProxyNotSetErr
	}
	p, err := url.Parse(cnProxy)
	if err != nil {
		return nil, fmt.Errorf("bad CNProxy: %v", err)
	}
	dialer, err := proxy.FromURL(p, proxy.Direct)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CNProxy: %v", err)
	}
	return dialer, nil
}
