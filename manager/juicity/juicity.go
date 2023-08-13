package juicity

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/daeuniverse/softwind/netproxy"
	"github.com/daeuniverse/softwind/protocol"
	"github.com/daeuniverse/softwind/protocol/direct"
	"github.com/daeuniverse/softwind/protocol/juicity"
	johnCommon "github.com/e14914c0-6759-480d-be89-66b7b7676451/BitterJohn/common"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/BitterJohn/server"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/common"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/manager"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/model"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/pkg/log"
	jsoniter "github.com/json-iterator/go"
)

func init() {
	manager.Register(string(protocol.ProtocolJuicity), New)
}

type Juicity struct {
	protocol protocol.Protocol
	dialer   *juicity.Dialer
	arg      *manager.ManageArgument
}

func New(dialer manager.Dialer, arg manager.ManageArgument) (manager.Manager, error) {
	var d netproxy.Dialer
	if dialer == nil {
		d = direct.SymmetricDirect
	} else {
		d = dialer
	}
	pinnedHash, err := base64.URLEncoding.DecodeString(common.SimplyGetParam(arg.Method, "pinned_certchain_sha256"))
	if err != nil {
		return nil, fmt.Errorf("failed to decode PinnedCertchainSha256")
	}
	d, err = server.NewDialer("juicity", dialer, &protocol.Header{
		ProxyAddress: net.JoinHostPort(arg.Host, arg.Port),
		SNI:          "",
		Feature1:     "bbr",
		TlsConfig: &tls.Config{
			NextProtos:         []string{"h3"},
			MinVersion:         tls.VersionTLS13,
			ServerName:         server.JuicityDomain,
			InsecureSkipVerify: true,
			VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
				if !bytes.Equal(johnCommon.GenerateCertChainHash(rawCerts), pinnedHash) {
					return fmt.Errorf("pinned hash of cert chain does not match")
				}
				return nil
			},
		},
		Cipher:   "",
		User:     arg.Argument.Username,
		Password: arg.Password,
		IsClient: true,
		Flags:    0,
	})
	if err != nil {
		return nil, err
	}

	return &Juicity{
		dialer:   d.(*juicity.Dialer),
		protocol: protocol.ProtocolJuicity,
		arg:      &arg,
	}, nil
}

func (s *Juicity) GetTurn(ctx context.Context, cmd protocol.MetadataCmd, body []byte) (respBody *manager.ReaderCloser, err error) {
	if len(body) >= 1<<17 {
		log.Trace("GetTurn(juicity): to: %v, len(body): %v", net.JoinHostPort(s.arg.Host, s.arg.Port), len(body))
	}

	conn, err := s.dialer.DialCmdMsg(cmd)
	if err != nil {
		return nil, err
	}
	go func() {
		<-ctx.Done()
		conn.SetDeadline(time.Now())
	}()

	req := make([]byte, len(body)+4)
	binary.BigEndian.PutUint32(req, uint32(len(body)))
	copy(req[4:], body)
	if _, err = conn.Write(req); err != nil {
		conn.Close()
		return nil, err
	}
	// reuse the req variable to read length
	if _, err = io.ReadFull(conn, req[:4]); err != nil {
		conn.Close()
		return nil, err
	}
	return &manager.ReaderCloser{Reader: io.LimitReader(conn, int64(binary.BigEndian.Uint32(req[:4]))), Closer: conn}, nil
}

func (s *Juicity) Ping(ctx context.Context) (resp *model.PingResp, err error) {
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

func (s *Juicity) SyncPassages(ctx context.Context, passages []model.Passage) (err error) {
	body, err := jsoniter.Marshal(passages)
	if err != nil {
		return err
	}
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
