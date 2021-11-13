package vmess

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	johnLog "github.com/e14914c0-6759-480d-be89-66b7b7676451/BitterJohn/pkg/log"
	protocol "github.com/e14914c0-6759-480d-be89-66b7b7676451/BitterJohn/server"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/BitterJohn/server/vmess"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/config"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/manager"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/model"
	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"io"
	"net"
	"time"
)

func init() {
	manager.Register("vmess", New)

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
	dialer manager.Dialer
	arg    manager.ManageArgument
	cmdKey []byte
}

func New(dialer manager.Dialer, arg manager.ManageArgument) (manager.Manager, error) {
	id, err := uuid.Parse(arg.Argument.Password)
	if err != nil {
		return nil, err
	}
	return &VMess{
		dialer: dialer,
		arg:    arg,
		cmdKey: vmess.NewID(id).CmdKey(),
	}, nil
}

func (s *VMess) GetTurn(ctx context.Context, cmd protocol.MetadataCmd, body []byte) (resp []byte, err error) {
	dialer := s.dialer
	if dialer == nil {
		dialer = &net.Dialer{}
	}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(s.arg.Host, s.arg.Port))
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	conn, err = vmess.NewConn(conn, vmess.Metadata{
		Type:     vmess.MetadataTypeMsg,
		Cmd:      cmd,
		InsCmd:   vmess.InstructionCmdTCP,
		Cipher:   vmess.CipherAES128GCM,
		IsClient: true,
	}, s.cmdKey)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	go func() {
		<-ctx.Done()
		conn.SetDeadline(time.Now())
	}()
	req := make([]byte, len(body)+4)
	binary.BigEndian.PutUint32(req, uint32(len(body)))
	copy(req[4:], body)
	if _, err = conn.Write(req); err != nil {
		return nil, err
	}

	// reuse the req variable to read length
	if _, err = io.ReadFull(conn, req[:4]); err != nil {
		return nil, err
	}
	resp = make([]byte, binary.BigEndian.Uint32(req[:4]))
	if _, err = io.ReadFull(conn, resp[:]); err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *VMess) Ping(ctx context.Context) (resp []byte, err error) {
	resp, err = s.GetTurn(ctx, protocol.MetadataCmdPing, []byte("ping"))
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *VMess) SyncPassages(ctx context.Context, passages []model.Passage) (err error) {
	body, err := jsoniter.Marshal(passages)
	if err != nil {
		return err
	}
	resp, err := s.GetTurn(ctx, protocol.MetadataCmdSyncPassages, body)
	if err != nil {
		return err
	}
	if !bytes.Equal(resp, []byte("OK")) {
		return fmt.Errorf("unexpected SyncPassages response from server: %v", string(resp))
	}
	return nil
}
