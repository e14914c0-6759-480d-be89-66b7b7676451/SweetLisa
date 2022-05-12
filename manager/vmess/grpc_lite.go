package vmess

import (
	"context"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/BitterJohn/transport/grpc_lite"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/manager"
	"net"
)

type GrpcLiteDialer struct {
	Dialer manager.Dialer
	Link   string
}

func (d *GrpcLiteDialer) Dial(network string, address string) (net.Conn, error) {
	//log.Debug("GrpcLiteDialer.Dial: %v %v, %v", d.Link, network, address)
	g, err := grpc_lite.NewGrpc(d.Link, d.Dialer)
	if err != nil {
		return nil, err
	}
	return g.Dial(network, address)
}

func (d *GrpcLiteDialer) DialContext(ctx context.Context, network string, address string) (net.Conn, error) {
	dc := manager.DialerConverter{Dialer: d}
	return dc.DialContext(ctx, network, address)
}
