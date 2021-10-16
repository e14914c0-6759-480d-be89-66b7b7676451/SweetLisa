package manager

import (
	"context"
	"fmt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/model"
	"strconv"
)

type ManageArgument struct {
	Host string
	Port string
	model.Argument
}

type Manager interface {
	Ping(ctx context.Context) (err error)
	SyncKeys(ctx context.Context, keys []model.Argument) (err error)
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
