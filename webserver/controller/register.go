package controller

import (
	"context"
	"fmt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/common"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/config"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/model"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/pkg/log"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/pkg/resolver"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/service"
	"github.com/gin-gonic/gin"
	"net"
	"net/netip"
	"strings"
	"time"
)

func hostsValidator(str string) error {
	if len(str) == 0 {
		return fmt.Errorf("host length cannot be zero")
	}
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
		if !req.Argument.Protocol.Valid() {
			log.Debug("Register: bad request: %v", req)
		}
		common.ResponseBadRequestError(c)
		return
	}
	// verify the server ticket
	ticObj := c.MustGet("TicketObj").(*model.Ticket)
	switch ticObj.Type {
	case model.TicketTypeServer, model.TicketTypeRelay:
	default:
		common.ResponseBadRequestError(c)
		return
	}
	go func(req model.Server, chatIdentifier string) {
		conf := config.GetConfig()
		if req.Argument.Protocol.WithTLS() {
			// waiting for the record
			domain, err := common.HostToSNI(model.GetFirstHost(req.Hosts), conf.Host)
			if err != nil {
				log.Error("%v", err)
			}
			log.Info("TLS SNI is %v", domain)

			log.Info("Waiting for DNS record")
			t := time.Now()
			for {
				ips, _ := resolver.LookupHost(domain)
				if len(ips) > 0 {
					break
				}
				if time.Since(t) > time.Minute {
					log.Error("timeout for waiting for DNS record")
				}
				time.Sleep(500 * time.Millisecond)
			}
			log.Info("Found DNS record")
		}
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
		log.Trace("ping %v use %v [%v; %v]", req.Name, req.Argument, req.Hosts, req.Port)
		if _, err = service.Ping(ctx, req); err != nil {
			err = fmt.Errorf("unreachable: %w", err)
			return
		}
		// register
		if err = service.RegisterServer(nil, req); err != nil {
			return
		}
		if err = service.ReqSyncPassagesByServer(nil, req.Ticket, false); err != nil {
			return
		}
	}(req, ticObj.ChatIdentifier)

	// assign subdomain for tls
	if conf := config.GetConfig(); conf.NameserverName != "" && conf.NameserverToken != "" && req.Argument.Protocol.WithTLS() {
		host := model.GetFirstHost(req.Hosts)
		if ip, e := netip.ParseAddr(host); e == nil {
			if e = service.AssignSubDomain(ip); e != nil {
				log.Warn("failed to assign subdomain: %v", e)
			}
		}
	}
	log.Info("Received a register request from %v: Chat: %v, Type: %v, Protocol: %v", req.Name, ticObj.ChatIdentifier, ticObj.Type, req.Argument.Protocol)
	passages := service.GetPassagesByServer(nil, req.Ticket)
	common.ResponseSuccess(c, passages)
}
