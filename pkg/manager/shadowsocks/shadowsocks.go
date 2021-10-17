package shadowsocks

import (
	"bytes"
	"context"
	"fmt"
	ss "github.com/e14914c0-6759-480d-be89-66b7b7676451/BitterJohn/server/shadowsocks"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/model"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/pkg/manager"
	jsoniter "github.com/json-iterator/go"
	"net"
	"time"
)

func init() {
	manager.Register("shadowsocks", New)
}

type Shadowsocks struct {
	arg        manager.ManageArgument
	masterKey  []byte
	cipherConf ss.CipherConf
}

func New(arg manager.ManageArgument) manager.Manager {
	cipherConf := ss.CiphersConf[string(arg.Argument.Protocol)]
	masterKey := ss.EVPBytesToKey(arg.Argument.Password, cipherConf.KeyLen)
	return &Shadowsocks{
		arg:        arg,
		masterKey:  masterKey,
		cipherConf: cipherConf,
	}
}

func (s *Shadowsocks) GetTurn(ctx context.Context, addr ss.Metadata, body []byte) (resp []byte, err error) {
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(s.arg.Host, s.arg.Port))
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	crw := ss.NewSSConn(conn, s.cipherConf, s.masterKey)
	go func() {
		<-ctx.Done()
		crw.SetDeadline(time.Now())
	}()
	return crw.GetTurn(addr, body)
}

func (s *Shadowsocks) Ping(ctx context.Context) (err error) {
	resp, err := s.GetTurn(ctx, ss.Metadata{Cmd: ss.MetadataCmdPing}, []byte("ping"))
	if err != nil {
		return err
	}
	if !bytes.Equal(resp, []byte("pong")) {
		return fmt.Errorf("unexpected ping response from server: %v", string(resp))
	}
	return nil
}

func (s *Shadowsocks) SyncKeys(ctx context.Context, keys []model.Argument) (err error) {
	body, err := jsoniter.Marshal(keys)
	if err != nil {
		return err
	}
	resp, err := s.GetTurn(ctx, ss.Metadata{Cmd: ss.MetadataCmdSyncKeys}, body)
	if err != nil {
		return err
	}
	if !bytes.Equal(resp, []byte("OK")) {
		return fmt.Errorf("unexpected SyncKeys response from server: %v", string(resp))
	}
	return nil
}
