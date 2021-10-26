package controller

import (
	"context"
	"fmt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/common"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/model"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/pkg/log"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/service"
	"github.com/gin-gonic/gin"
	"net"
	"strings"
	"time"
)

func hostsValidator(str string) error {
	hosts := strings.Split(str, ",")
	for _, host := range hosts {
		if err := hostValidator(host); err != nil {
			return fmt.Errorf("%v: %w", host, err)
		}
	}
	return nil
}

func hostValidator(str string) error {
	e := fmt.Errorf("Invalid Host")
	if net.ParseIP(str) != nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupHost(ctx, str)
	if err != nil {
		return e
	}
	if len(addrs) == 0 {
		return e
	}
	return nil
}

// PostRegister registers a server
func PostRegister(c *gin.Context) {
	var req model.Server
	if err := c.ShouldBindJSON(&req); err != nil || c.Param("Ticket") != req.Ticket {
		common.ResponseBadRequestError(c)
		return
	}
	// required info
	if req.Hosts == "" ||
		req.Port == 0 ||
		!req.Argument.Protocol.Valid() ||
		req.Name == "" ||
		hostsValidator(req.Hosts) != nil {
		common.ResponseBadRequestError(c)
		return
	}
	// verify the server ticket
	ticObj, err := service.GetValidTicketObj(nil, req.Ticket)
	if err != nil {
		common.ResponseError(c, err)
		return
	}
	switch ticObj.Type {
	case model.TicketTypeServer, model.TicketTypeRelay:
	default:
		common.ResponseBadRequestError(c)
		return
	}
	go func(req model.Server, chatIdentifier string) {
		// waiting for the starting of BitterJohn
		time.Sleep(5 * time.Second)
		var err error

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		defer func() {
			if err != nil {
				log.Warn("reject to register %v: %v", req.Name, err)
			} else {
				log.Info("register %v successfully", req.Name)
			}
		}()
		// ping test
		log.Trace("ping %v use %v", req.Name, req.Argument)
		if err = service.Ping(ctx, req); err != nil {
			err = fmt.Errorf("unreachable: %w", err)
			return
		}
		// register
		if err = service.RegisterServer(nil, req); err != nil {
			return
		}
		if ticObj.Type == model.TicketTypeRelay {
			if err = service.ReqSyncPassagesByChatIdentifier(nil, chatIdentifier, false); err != nil {
				return
			}
		}
	}(req, ticObj.ChatIdentifier)
	log.Info("Received a register request from %v: Chat: %v, Type: %v", req.Name, ticObj.ChatIdentifier, ticObj.Type)
	passages := service.GetPassagesByServer(nil, req.Ticket)
	common.ResponseSuccess(c, passages)
}
