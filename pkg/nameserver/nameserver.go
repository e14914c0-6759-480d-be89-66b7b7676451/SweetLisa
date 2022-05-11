package nameserver

import (
	"context"
	"fmt"
)

type Nameserver interface {
	Assign(ctx context.Context, domain string, ip string) error
	RemoveRecords(ctx context.Context, domain string) error
}

type Creator func(token string) (Nameserver, error)

var creatorMapping = make(map[string]Creator)

func Register(name string, creator Creator) {
	creatorMapping[name] = creator
}

func NewNameserver(name string, token string) (ns Nameserver, err error) {
	creator, ok := creatorMapping[name]
	if !ok {
		return nil, fmt.Errorf("unexpected name: %v", name)
	}
	return creator(token)
}
