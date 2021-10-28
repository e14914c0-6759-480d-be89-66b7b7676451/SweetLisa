package service

import (
	"context"
	"fmt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/manager"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/model"
	jsoniter "github.com/json-iterator/go"
	"strconv"
)

func Ping(ctx context.Context, server model.Server) (*model.PingResp, error) {
	mng, err := manager.NewManager(ChooseDialer(server), manager.ManageArgument{
		Host:     model.GetFirstHost(server.Hosts),
		Port:     strconv.Itoa(server.Port),
		Argument: server.Argument,
	})
	if err != nil {
		return nil, fmt.Errorf("NewManager(%v): %w", server.Name, err)
	}
	var bResp []byte
	if bResp, err = mng.Ping(ctx); err != nil {
		return nil, fmt.Errorf("Ping: %w", err)
	}
	var r model.PingResp
	if err = jsoniter.Unmarshal(bResp, &r); err != nil {
		return nil, err
	}
	return &r, nil
}
