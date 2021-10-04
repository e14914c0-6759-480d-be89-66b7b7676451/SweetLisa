package controller

import (
	"fmt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/common"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/model"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/service"
	"github.com/gin-gonic/gin"
	gonanoid "github.com/matoous/go-nanoid"
)

// GetTicket will add a ticket to database or renew a ticket existing
func GetTicket(ctx *gin.Context) {
	verificationCode := ctx.Query("Verification")
	chatIdentifier := ctx.GetString("ChatIdentifier")
	if err := service.Verified(verificationCode, chatIdentifier); err != nil {
		common.ResponseError(ctx, err)
		return
	}
	ticket, err := gonanoid.Nanoid(model.TicketLength)
	if err != nil {
		common.ResponseError(ctx, fmt.Errorf("%v: try again please", err))
		return
	}
	tic, err := service.SaveTicket(ticket, chatIdentifier)
	if err != nil {
		common.ResponseError(ctx, err)
		return
	}
	common.ResponseSuccess(ctx, gin.H{
		"Ticket": tic,
	})
}
