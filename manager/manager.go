package manager

import (
	"context"
	"fmt"
	"io"
	"strconv"

	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/model"
)

type ManageArgument struct {
	Host       string
	Port       string
	RootDomain string // this is for TLS SNI
	model.Argument
}

type Manager interface {
	Ping(ctx context.Context) (resp *model.PingResp, err error)
	SyncPassages(ctx context.Context, passages []model.Passage) (err error)
}

type ReaderCloser struct {
	Reader io.Reader
	Closer io.Closer
}

type Creator func(conn Dialer, arg ManageArgument) (Manager, error)

var Mapper = make(map[string]Creator)

func Register(name string, c Creator) {
	Mapper[name] = c
}

func NewManager(dialer Dialer, arg ManageArgument) (Manager, error) {
	creator, ok := Mapper[string(arg.Protocol)]
	if !ok {
		return nil, fmt.Errorf("no manager creator registered for %v", strconv.Quote(string(arg.Protocol)))
	}
	return creator(dialer, arg)
}
