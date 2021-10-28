package controller

import (
	"fmt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/common"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/model"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/pkg/log"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/service"
	"github.com/gin-gonic/gin"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

func NameToShow(server model.Server) string {
	remaining := make([]int64, 0, 3)
	if server.BandwidthLimit.TotalLimitGiB > 0 {
		remaining = append(remaining, server.BandwidthLimit.TotalLimitGiB*1024*1024-
			(server.BandwidthLimit.UplinkKiB-server.BandwidthLimit.UplinkInitialKiB)-
			(server.BandwidthLimit.DownlinkKiB-server.BandwidthLimit.DownlinkInitialKiB))
	}
	if server.BandwidthLimit.UplinkLimitGiB > 0 {
		remaining = append(remaining, server.BandwidthLimit.UplinkLimitGiB*1024*1024-
			(server.BandwidthLimit.UplinkKiB-server.BandwidthLimit.UplinkInitialKiB))
	}
	if server.BandwidthLimit.DownlinkLimitGiB > 0 {
		remaining = append(remaining, server.BandwidthLimit.DownlinkLimitGiB*1024*1024-
			(server.BandwidthLimit.DownlinkKiB-server.BandwidthLimit.DownlinkInitialKiB))
	}
	if len(remaining) == 0 {
		return server.Name
	}
	sort.Slice(remaining, func(i, j int) bool {
		return remaining[i] < remaining[j]
	})
	if remaining[0] < 0 {
		remaining[0] = 0
	}
	return fmt.Sprintf("[%.1f GiB] %v", float64(remaining[0])/1024/1024, server.Name)
}

// GetSubscription returns the user's subscription
func GetSubscription(c *gin.Context) {
	ticket := c.Param("Ticket")
	// verify the ticket
	ticObj, err := service.GetValidTicketObj(nil, ticket)
	if err != nil {
		common.ResponseError(c, err)
		return
	}
	switch ticObj.Type {
	case model.TicketTypeUser, model.TicketTypeRelay:
	default:
		common.ResponseBadRequestError(c)
		return
	}
	// get servers
	servers, err := service.GetServersByChatIdentifier(nil, ticObj.ChatIdentifier, true)
	if err != nil {
		common.ResponseError(c, err)
		return
	}
	sort.Slice(servers, func(i, j int) bool {
		return servers[i].Name < servers[j].Name
	})
	// generate keys
	var sip008 = model.SIP008{
		Version: 1,
	}
	sip008.Servers = append(sip008.Servers, model.SIP008Server{
		Id:         "00000000-0000-0000-0000-000000000000",
		Remarks:    fmt.Sprintf("ExpireAt: %v", ticObj.ExpireAt.Format("2006-01-02 15:04 MST")),
		Server:     "127.0.0.1",
		ServerPort: 1024,
		Password:   "0",
		Method:     "chacha20-ietf-poly1305",
	})
	var relays []model.Server
	var svrs []model.Server
	for _, server := range servers {
		if server.FailureCount >= model.MaxFailureCount {
			// do not return lost-alive server
			continue
		}
		svrTic, err := service.GetValidTicketObj(nil, server.Ticket)
		if err != nil {
			log.Warn("GetSubscription: GetValidTicketObj: %v", err)
			continue
		}
		if svrTic.Type == model.TicketTypeRelay {
			relays = append(relays, server)
			continue
		}
		if svrTic.Type == model.TicketTypeServer {
			svrs = append(svrs, server)
		}
		arg := model.GetUserArgument(server.Ticket, ticket)
		for j, host := range strings.Split(server.Hosts, ",") {
			var id string
			if j == 0 {
				id = common.StringToUUID5(arg.Password)
			} else {
				id = common.StringToUUID5(arg.Password + ":" + strconv.Itoa(j))
			}
			sip008.Servers = append(sip008.Servers, model.SIP008Server{
				Id:         id,
				Remarks:    NameToShow(server),
				Server:     host,
				ServerPort: server.Port,
				Password:   arg.Password,
				Method:     arg.Method,
			})
		}
	}
	sort.Slice(relays, func(i, j int) bool {
		return relays[i].Name < relays[j].Name
	})
	for _, relay := range relays {
		for _, svr := range svrs {
			arg := model.GetRelayUserArgument(svr.Ticket, relay.Ticket, ticket)
			for j, host := range strings.Split(relay.Hosts, ",") {
				var id string
				if j == 0 {
					id = common.StringToUUID5(arg.Password)
				} else {
					id = common.StringToUUID5(arg.Password + ":" + strconv.Itoa(j))
				}
				sip008.Servers = append(sip008.Servers, model.SIP008Server{
					Id:         id,
					Remarks:    fmt.Sprintf("%v -> %v", NameToShow(relay), NameToShow(svr)),
					Server:     host,
					ServerPort: relay.Port,
					Password:   arg.Password,
					Method:     arg.Method,
				})
			}
		}
	}
	c.JSON(http.StatusOK, sip008)
}
