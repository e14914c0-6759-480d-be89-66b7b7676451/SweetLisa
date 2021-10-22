package shadowsocks

import (
	"bytes"
	"context"
	"fmt"
	johnLog "github.com/e14914c0-6759-480d-be89-66b7b7676451/BitterJohn/pkg/log"
	ss "github.com/e14914c0-6759-480d-be89-66b7b7676451/BitterJohn/server/shadowsocks"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/config"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/model"
	jsoniter "github.com/json-iterator/go"
	"net"
	"time"
)

func init() {
	model.Register("shadowsocks", New)

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
	arg        model.ManageArgument
	masterKey  []byte
	cipherConf ss.CipherConf
}

func New(arg model.ManageArgument) model.Manager {
	cipherConf := ss.CiphersConf[arg.Argument.Method]
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
	crw, err := ss.NewSSConn(conn, s.cipherConf, s.masterKey)
	if err != nil {
		conn.Close()
		return nil, err
	}
	defer crw.Close()
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

func (s *Shadowsocks) SyncPassages(ctx context.Context, passages []model.Passage) (err error) {
	body, err := jsoniter.Marshal(passages)
	if err != nil {
		return err
	}
	resp, err := s.GetTurn(ctx, ss.Metadata{Cmd: ss.MetadataCmdSyncPassages}, body)
	if err != nil {
		return err
	}
	if !bytes.Equal(resp, []byte("OK")) {
		return fmt.Errorf("unexpected SyncPassages response from server: %v", string(resp))
	}
	return nil
}
