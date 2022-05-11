package vmess

import (
	"context"
	"github.com/Qv2ray/gun/pkg/proto"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/BitterJohn/pool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"net"
	"sync"
	"time"
)

type GrpcConn struct {
	tun    proto.GunService_TunClient
	mu     sync.Mutex // mu protects reading
	buf    []byte
	offset int
}

func (c *GrpcConn) Read(p []byte) (n int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.buf != nil {
		n = copy(p, c.buf[c.offset:])
		c.offset += n
		if c.offset == len(c.buf) {
			pool.Put(c.buf)
			c.buf = nil
		}
		return n, nil
	}
	recv, err := c.tun.Recv()
	if err != nil {
		return 0, err
	}
	n = copy(p, recv.Data)
	c.buf = pool.Get(len(recv.Data) - n)
	copy(c.buf, recv.Data[n:])
	c.offset = 0
	return n, nil
}

func (c *GrpcConn) Write(p []byte) (n int, err error) {
	return len(p), c.tun.Send(&proto.Hunk{Data: p})
}

func (c *GrpcConn) Close() error {
	return nil
}
func (c *GrpcConn) LocalAddr() net.Addr {
	// FIXME
	return nil
}
func (c *GrpcConn) RemoteAddr() net.Addr {
	p, _ := peer.FromContext(c.tun.Context())
	return p.Addr
}

// SetDeadline is not implemented
func (c *GrpcConn) SetDeadline(t time.Time) error {
	return nil
}

// SetReadDeadline is not implemented
func (c *GrpcConn) SetReadDeadline(t time.Time) error {
	return nil
}

// SetWriteDeadline is not implemented
func (c *GrpcConn) SetWriteDeadline(t time.Time) error {
	return nil
}

type GrpcDialer struct {
}

func (d *GrpcDialer) Dial(network string, address string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, address)
}
func (d *GrpcDialer) DialContext(ctx context.Context, network string, address string) (net.Conn, error) {
	cc, err := grpc.Dial(address)
	if err != nil {
		return nil, err
	}
	client := proto.NewGunServiceClient(cc)
	clientX := client.(proto.GunServiceClientX)
	tun, err := clientX.TunCustomName(ctx, "GunService")
	conn := GrpcConn{
		tun:    tun,
		mu:     sync.Mutex{},
		buf:    nil,
		offset: 0,
	}
	return &conn, nil

}
