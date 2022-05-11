package vmess

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	johnLog "github.com/e14914c0-6759-480d-be89-66b7b7676451/BitterJohn/pkg/log"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/BitterJohn/protocol"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/BitterJohn/protocol/vmess"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/common"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/config"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/manager"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/model"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/pkg/log"
	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"io"
	"net"
	"net/url"
	"time"
)

func init() {
	manager.Register(string(model.ProtocolVMessTCP), New)
	manager.Register(string(model.ProtocolVMessTlsGrpc), NewWithGrpc)

	// init the log of bitterJohnConfig with sweetLisa's config
	params := *config.GetConfig()
	var logFile string
	if params.LogFile != "" {
		logFile += ".bitterJohn"
	}
	logWay := "console"
	if params.LogFile != "" {
		logWay = "file"
	}
	johnLog.InitLog(logWay, params.LogFile, params.LogLevel, params.LogMaxDays, params.LogDisableColor, params.LogDisableTimestamp)
}

type VMess struct {
	protocol model.Protocol
	dialer   manager.Dialer
	arg      manager.ManageArgument
	cmdKey   []byte
}

func New(dialer manager.Dialer, arg manager.ManageArgument) (manager.Manager, error) {
	id, err := uuid.Parse(arg.Argument.Password)
	if err != nil {
		return nil, err
	}
	return &VMess{
		dialer:   dialer,
		arg:      arg,
		protocol: model.ProtocolVMessTCP,
		cmdKey:   vmess.NewID(id).CmdKey(),
	}, nil
}

func NewWithGrpc(dialer manager.Dialer, arg manager.ManageArgument) (manager.Manager, error) {
	id, err := uuid.Parse(arg.Argument.Password)
	if err != nil {
		return nil, err
	}
	return &VMess{
		dialer:   dialer,
		arg:      arg,
		protocol: model.ProtocolVMessTlsGrpc,
		cmdKey:   vmess.NewID(id).CmdKey(),
	}, nil
}

func (s *VMess) GetTurn(ctx context.Context, cmd protocol.MetadataCmd, body []byte) (respBody *manager.ReaderCloser, err error) {
	if len(body) >= 1<<17 {
		log.Trace("GetTurn(vmess): to: %v, len(body): %v", net.JoinHostPort(s.arg.Host, s.arg.Port), len(body))
	}
	addr := net.JoinHostPort(s.arg.Host, s.arg.Port)
	dialer := s.dialer
	if dialer == nil {
		dialer = &net.Dialer{}
	}
	if s.protocol == model.ProtocolVMessTlsGrpc {
		sni, err := common.HostToSNI(s.arg.Host, s.arg.RootDomain)
		if err != nil {
			return nil, err
		}
		u := url.URL{
			Scheme:   "grpc",
			Host:     addr,
			RawQuery: url.Values{"sni": []string{sni}}.Encode(),
		}
		dialer = &GrpcLiteDialer{Dialer: dialer, Link: u.String()}
	}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	vConn, err := vmess.NewConn(conn, vmess.Metadata{
		Metadata: protocol.Metadata{
			Type:     protocol.MetadataTypeMsg,
			Cmd:      cmd,
			Cipher:   string(vmess.CipherAES128GCM),
			IsClient: true,
		},
		Network: "tcp",
	}, s.cmdKey)
	if err != nil {
		conn.Close()
		return nil, err
	}
	go func() {
		<-ctx.Done()
		vConn.SetDeadline(time.Now())
	}()
	req := make([]byte, len(body)+4)
	binary.BigEndian.PutUint32(req, uint32(len(body)))
	copy(req[4:], body)
	if _, err = vConn.Write(req); err != nil {
		vConn.Close()
		return nil, err
	}

	// reuse the req variable to read length
	if _, err = io.ReadFull(vConn, req[:4]); err != nil {
		vConn.Close()
		return nil, err
	}
	return &manager.ReaderCloser{Reader: io.LimitReader(vConn, int64(binary.BigEndian.Uint32(req[:4]))), Closer: vConn}, nil
}

func (s *VMess) Ping(ctx context.Context) (resp *model.PingResp, err error) {
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

func (s *VMess) SyncPassages(ctx context.Context, passages []model.Passage) (err error) {
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
