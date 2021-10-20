package service

import (
	"context"
	"fmt"
	"github.com/boltdb/bolt"
	"strings"
	"sync"
)

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
