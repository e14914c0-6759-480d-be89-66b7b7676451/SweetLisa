package controller

import (
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/common"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/service"
	"github.com/gin-gonic/gin"
)

// PostTicket will add a ticket to database or renew a ticket existing
func PostTicket(ctx *gin.Context) {
	var data struct {
		BlindedTicket string
	}
	if err := ctx.ShouldBindJSON(&data); err != nil {
		common.ResponseBadRequestError(ctx)
		return
	}
	chatIdentifier := ctx.GetString("chatIdentifier")
	sigBytes, err := service.BlindSign(data.BlindedTicket, chatIdentifier)
	if err != nil {
		common.ResponseError(ctx, err)
		return
	}
	sig, err := service.SaveSig(sigBytes)
	if err != nil {
		common.ResponseError(ctx, err)
		return
	}
	common.ResponseSuccess(ctx, gin.H{
		"Sig": sig,
	})
}
