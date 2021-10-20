package model

import (
	"context"
	"fmt"
	"strconv"
)

type ManageArgument struct {
	Host string
	Port string
	Argument
}

type Manager interface {
	Ping(ctx context.Context) (err error)
	SyncPassages(ctx context.Context, passages []Passage) (err error)
}

type Creator func(arg ManageArgument) Manager

var Mapper = make(map[string]Creator)

func Register(name string, c Creator) {
	Mapper[name] = c
}

func NewManager(arg ManageArgument) (Manager, error) {
	creator, ok := Mapper[string(arg.Protocol)]
	if !ok {
		return nil, fmt.Errorf("no manager creator registered for %v", strconv.Quote(string(arg.Protocol)))
	}
	return creator(arg), nil
}
