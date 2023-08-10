package shadowsocks

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/daeuniverse/softwind/ciphers"
	"github.com/daeuniverse/softwind/common"
	"github.com/daeuniverse/softwind/netproxy"
	"github.com/daeuniverse/softwind/protocol"
	"github.com/daeuniverse/softwind/protocol/direct"
	ss "github.com/daeuniverse/softwind/protocol/shadowsocks"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/manager"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/model"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/pkg/log"
	jsoniter "github.com/json-iterator/go"
)

func init() {
	manager.Register(string(protocol.ProtocolShadowsocks), New)
}

type Shadowsocks struct {
	dialer     manager.Dialer
	arg        manager.ManageArgument
	masterKey  []byte
	cipherConf *ciphers.CipherConf
}

func New(dialer manager.Dialer, arg manager.ManageArgument) (manager.Manager, error) {
	cipherConf := ciphers.AeadCiphersConf[arg.Argument.Method]
	masterKey := common.EVPBytesToKey(arg.Argument.Password, cipherConf.KeyLen)
	return &Shadowsocks{
		dialer:     dialer,
		arg:        arg,
		masterKey:  masterKey,
		cipherConf: cipherConf,
	}, nil
}

func (s *Shadowsocks) GetTurn(ctx context.Context, cmd protocol.MetadataCmd, body []byte) (respBody *manager.ReaderCloser, err error) {
	if len(body) >= 1<<17 {
		log.Trace("GetTurn(ss): to: %v, len(body): %v", net.JoinHostPort(s.arg.Host, s.arg.Port), len(body))
	}
	dialer := s.dialer
	if dialer == nil {
		dialer = &netproxy.ContextDialerConverter{
			Dialer: direct.SymmetricDirect,
		}
	}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(s.arg.Host, s.arg.Port))
	if err != nil {
		return nil, err
	}
	crw, err := ss.NewTCPConn(conn, protocol.Metadata{
		Type:     protocol.MetadataTypeMsg,
		Cmd:      cmd,
		Cipher:   s.arg.Argument.Method,
		IsClient: true,
	}, s.masterKey, nil)
	if err != nil {
		conn.Close()
		return nil, err
	}
	go func() {
		<-ctx.Done()
		crw.SetDeadline(time.Now())
	}()
	if _, err = crw.Write(body); err != nil {
		crw.Close()
		return nil, err
	}

	metadata, err := crw.ReadMetadata()
	if err != nil {
		crw.Close()
		return nil, err
	}
	return &manager.ReaderCloser{Reader: io.LimitReader(crw, int64(metadata.LenMsgBody)), Closer: crw}, nil
}

func (s *Shadowsocks) Ping(ctx context.Context) (resp *model.PingResp, err error) {
	respBody, err := s.GetTurn(ctx, protocol.MetadataCmdPing, []byte("ping"))
	if err != nil {
		return nil, err
	}
	defer respBody.Closer.Close()
	var r model.PingResp
	if err = jsoniter.NewDecoder(respBody.Reader).Decode(&r); err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Shadowsocks) SyncPassages(ctx context.Context, passages []model.Passage) (err error) {
	body, err := jsoniter.Marshal(passages)
	if err != nil {
		return err
	}
	log.Trace("SyncPassages: to: %v, len(body): %v", s.arg.Host, len(body))
	respBody, err := s.GetTurn(ctx, protocol.MetadataCmdSyncPassages, body)
	if err != nil {
		return err
	}
	defer respBody.Closer.Close()
	var buf = make([]byte, 2)
	if _, err = io.ReadFull(respBody.Reader, buf); err != nil {
		return err
	}
	if !bytes.Equal(buf, []byte("OK")) {
		return fmt.Errorf("unexpected SyncPassages response from server: %v", string(buf))
	}
	return nil
}
