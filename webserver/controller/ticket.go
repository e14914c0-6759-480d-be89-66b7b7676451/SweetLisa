package controller

import (
	"fmt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/common"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/model"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/service"
	"github.com/gin-gonic/gin"
	gonanoid "github.com/matoous/go-nanoid"
)

// GetTicket will add a ticket to database
func GetTicket(c *gin.Context) {
	var query struct {
		Type             int
		VerificationCode string
	}
	if err := c.ShouldBindQuery(&query); err != nil ||
		!model.TicketType(query.Type).IsValid() {
		common.ResponseBadRequestError(c)
		return
	}
	chatIdentifier := c.Param("ChatIdentifier")
	if err := service.Verified(nil, query.VerificationCode, chatIdentifier); err != nil {
		common.ResponseError(c, err)
		return
	}
	ticket, err := gonanoid.Generate(common.Alphabet, model.TicketLength)
	if err != nil {
		common.ResponseError(c, fmt.Errorf("%v: try again please", err))
		return
	}
	// SaveTicket
	tic, err := service.SaveTicket(nil, ticket, model.TicketType(query.Type), chatIdentifier)
	if err != nil {
		common.ResponseError(c, err)
		return
	}
	// ReqSyncPassagesByChatIdentifier
	if err := service.ReqSyncPassagesByChatIdentifier(nil, chatIdentifier, true); err != nil {
		common.ResponseError(c, fmt.Errorf("ReqSyncPassagesByChatIdentifier: %v", err))
		return
	}
	common.ResponseSuccess(c, gin.H{
		"Ticket": tic,
	})
}

// PostRenew will renew a ticket existing
func PostRenew(c *gin.Context) {
	var req struct {
		VerificationCode string
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ResponseBadRequestError(c)
		return
	}
	ticket := c.Param("Ticket")
	ticObj, err := service.GetValidTicketObj(nil, ticket)
	if err != nil {
		common.ResponseError(c, err)
		return
	}
	// verify the VerificationCode
	if err := service.Verified(nil, req.VerificationCode, ticObj.ChatIdentifier); err != nil {
		common.ResponseError(c, err)
		return
	}
	if ticObj.Type != model.TicketTypeUser {
		common.ResponseBadRequestError(c)
		return
	}
	renewedTic, err := service.SaveTicket(nil, ticket, ticObj.Type, ticObj.ChatIdentifier)
	if err != nil {
		common.ResponseError(c, err)
		return
	}
	if common.Expired(ticObj.ExpireAt) {
		// ReqSyncPassagesByChatIdentifier
		if err := service.ReqSyncPassagesByChatIdentifier(nil, ticObj.ChatIdentifier, true); err != nil {
			common.ResponseError(c, err)
			return
		}
	}
	common.ResponseSuccess(c, renewedTic)
}
