package controller

import (
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/common"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/model"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/service"
	"github.com/gin-gonic/gin"
	"net/http"
)

// GetSubscription returns the user's subscription
func GetSubscription(c *gin.Context) {
	ticket := c.Param("Ticket")
	// verify the ticket
	ticObj, err := service.GetValidTicketObj(ticket)
	if err != nil {
		common.ResponseError(c, err)
		return
	}
	chatIdentifier := c.Param("ChatIdentifier")
	if ticObj.ChatIdentifier != chatIdentifier {
		common.ResponseBadRequestError(c)
		return
	}
	// get servers
	servers, err := service.GetServersByChatIdentifier(chatIdentifier)
	if err != nil {
		common.ResponseError(c, err)
		return
	}
	// generate keys
	var sip008 = model.SIP008{
		Version: 1,
	}
	for _, svr := range servers {
		arg := svr.GetUserArgument(ticket)
		sip008.Servers = append(sip008.Servers, model.SIP008Server{
			Id:         common.StringToUUID5(svr.Ticket),
			Remarks:    svr.Name,
			Server:     svr.Host,
			ServerPort: svr.Port,
			Password:   arg.Password,
			Method:     arg.Method,
		})
	}
	c.JSON(http.StatusOK, sip008)
}
