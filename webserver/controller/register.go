package controller

import (
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/common"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/model"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/pkg/log"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/service"
	"github.com/gin-gonic/gin"
)

// PostRegister registers a server
func PostRegister(ctx *gin.Context) {
	var req model.Server
	if err := ctx.ShouldBindJSON(&req); err != nil || ctx.Param("Ticket") != req.Ticket {
		common.ResponseBadRequestError(ctx)
		return
	}
	// required info
	if req.Host == "" ||
		req.Port == 0 ||
		!req.Argument.Protocol.Valid() ||
		req.Name == "" {
		common.ResponseBadRequestError(ctx)
		return
	}
	// verify the server ticket
	ticObj, err := service.GetValidTicketObj(nil, req.Ticket)
	if err != nil {
		common.ResponseError(ctx, err)
		return
	}
	chatIdentifier := ctx.Param("ChatIdentifier")
	if ticObj.ChatIdentifier != chatIdentifier {
		common.ResponseBadRequestError(ctx)
		return
	}
	switch ticObj.Type {
	case model.TicketTypeServer, model.TicketTypeRelay:
	default:
		common.ResponseBadRequestError(ctx)
		return
	}
	// register
	if err := service.RegisterServer(nil, req); err != nil {
		common.ResponseError(ctx, err)
		return
	}
	log.Info("Received a register request from %v: Chat: %v, Name: %v, Type: %v", ctx.ClientIP(), req.Name, chatIdentifier, ticObj.Type)
	keys := service.GetKeysByServer(nil, req)
	log.Trace("register: %v", keys)
	common.ResponseSuccess(ctx, keys)
}
