package shadowsocks

import (
	"bytes"
	"context"
	"fmt"
	johnLog "github.com/e14914c0-6759-480d-be89-66b7b7676451/BitterJohn/pkg/log"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/BitterJohn/protocol"
	ss "github.com/e14914c0-6759-480d-be89-66b7b7676451/BitterJohn/protocol/shadowsocks"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/config"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/manager"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/model"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/pkg/log"
	jsoniter "github.com/json-iterator/go"
	"io"
	"net"
	"time"
)

func init() {
	manager.Register("shadowsocks", New)

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

type Shadowsocks struct {
	dialer     manager.Dialer
	arg        manager.ManageArgument
	masterKey  []byte
	cipherConf ss.CipherConf
}

func New(dialer manager.Dialer, arg manager.ManageArgument) (manager.Manager, error) {
	cipherConf := ss.CiphersConf[arg.Argument.Method]
	masterKey := ss.EVPBytesToKey(arg.Argument.Password, cipherConf.KeyLen)
	return &Shadowsocks{
		dialer:     dialer,
		arg:        arg,
		masterKey:  masterKey,
		cipherConf: cipherConf,
	}, nil
}

func (s *Shadowsocks) GetTurn(ctx context.Context, cmd protocol.MetadataCmd, body []byte) (resp []byte, err error) {
	if len(body) >= 1<<17 {
		log.Trace("GetTurn(ss): to: %v, len(body): %v", net.JoinHostPort(s.arg.Host, s.arg.Port), len(body))
	}
	dialer := s.dialer
	if dialer == nil {
		dialer = &net.Dialer{}
	}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(s.arg.Host, s.arg.Port))
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	crw, err := ss.NewTCPConn(conn, protocol.Metadata{
		Type:     protocol.MetadataTypeMsg,
		Cmd:      cmd,
		Network:  "tcp",
		Cipher:   s.arg.Argument.Method,
		IsClient: false,
	}, s.masterKey, nil)

	if err != nil {
		return nil, err
	}
	defer crw.Close()
	go func() {
		<-ctx.Done()
		crw.SetDeadline(time.Now())
	}()
	if _, err = crw.Write(body); err != nil {
		return nil, err
	}

	metadata, err := crw.ReadMetadata()
	if err != nil {
		return nil, err
	}
	resp = make([]byte, metadata.LenMsgBody&0xffffff)
	if _, err := io.ReadFull(crw, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *Shadowsocks) Ping(ctx context.Context) (resp []byte, err error) {
	resp, err = s.GetTurn(ctx, protocol.MetadataCmdPing, []byte("ping"))
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *Shadowsocks) SyncPassages(ctx context.Context, passages []model.Passage) (err error) {
	body, err := jsoniter.Marshal(passages)
	if err != nil {
		return err
	}
	//log.Trace("SyncPassages: to: %v, len(body): %v", s.arg.Host, len(body))
	resp, err := s.GetTurn(ctx, protocol.MetadataCmdSyncPassages, body)
	if err != nil {
		return err
	}
	if !bytes.Equal(resp, []byte("OK")) {
		return fmt.Errorf("unexpected SyncPassages response from server: %v", string(resp))
	}
	return nil
}
