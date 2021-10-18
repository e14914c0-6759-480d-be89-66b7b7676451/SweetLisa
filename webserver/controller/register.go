package controller

import (
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/common"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/model"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/pkg/log"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/service"
	"github.com/gin-gonic/gin"
	"time"
)

// PostRegister registers a server
func PostRegister(ctx *gin.Context) {
	var req model.Server
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.ResponseBadRequestError(ctx)
		return
	}
	// required info
	if req.Host == "" ||
		req.Port == 0 ||
		!req.ManageArgument.Protocol.Valid() ||
		req.Name == "" {
		common.ResponseBadRequestError(ctx)
		return
	}
	// verify the server ticket
	code, err := service.GetValidTicketObj(req.Ticket)
	if err != nil {
		common.ResponseError(ctx, err)
		return
	}
	chatIdentifier := ctx.Param("ChatIdentifier")
	if code.ChatIdentifier != chatIdentifier {
		common.ResponseBadRequestError(ctx)
		return
	}
	if code.Type != model.TicketTypeServer {
		common.ResponseBadRequestError(ctx)
		return
	}
	// register
	req.FailureCount = 0
	req.LastSeen = time.Now()
	if err := service.RegisterServer(req); err != nil {
		common.ResponseError(ctx, err)
		return
	}
	log.Info("Received a register request from %v: Chat: %v, Name: %v", ctx.ClientIP(), req.Name, chatIdentifier)
	keys := service.GetKeysByServer(req)
	log.Trace("keys: %v", keys)
	common.ResponseSuccess(ctx, keys)
}
