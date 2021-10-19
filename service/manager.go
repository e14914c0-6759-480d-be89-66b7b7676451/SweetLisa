package service

import (
	"context"
	"fmt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/model"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/pkg/log"
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
		Argument: server.Argument,
	})
	return mng.SyncKeys(ctx, keys)
}

func SyncKeysByChatIdentifier(ctx context.Context, chatIdentifier string) (err error) {
	servers, err := GetServersByChatIdentifier(chatIdentifier)
	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	var errs []string
	var mu sync.Mutex
	for _, svr := range servers {
		keys := GetKeysByServer(svr)
		wg.Add(1)
		go func(svr model.Server, keys []model.Server) {
			log.Trace("SyncKeysByChatIdentifier: chat: %v, svr: %v, keys: %v", chatIdentifier, svr, keys)
			defer wg.Done()
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
			if err = mng.SyncKeys(ctx, keys); err != nil {
				mu.Lock()
				errs = append(errs, "SyncKeys: "+err.Error())
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
