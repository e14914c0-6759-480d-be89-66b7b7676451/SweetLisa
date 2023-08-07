package manager

import (
	"context"

	"github.com/daeuniverse/softwind/netproxy"
	"golang.org/x/net/proxy"
)

// Dialer is used to create connection.
type Dialer interface {
	// DialContext connects to the given address
	DialContext(ctx context.Context, network, addr string) (c netproxy.Conn, err error)
}

type DialerConverter struct {
	Dialer proxy.Dialer
}

func (d *DialerConverter) DialContext(ctx context.Context, network, addr string) (c netproxy.Conn, err error) {
	var done = make(chan struct{})
	go func() {
		c, err = d.Dialer.Dial(network, addr)
		if err != nil {
			return
		}
		select {
		case <-ctx.Done():
			_ = c.Close()
		default:
			close(done)
		}
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-done:
		return c, err
	}
}

func (d *DialerConverter) Dial(network, addr string) (c netproxy.Conn, err error) {
	return d.Dialer.Dial(network, addr)
}
