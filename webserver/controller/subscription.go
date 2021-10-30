package controller

import (
	"fmt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/common"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/model"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/pkg/log"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/service"
	"github.com/gin-gonic/gin"
	"net"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const PasswordReserve = "#SWEETLISA#"

func NameToShow(server model.Server) string {
	remaining := make([]int64, 0, 3)
	if server.BandwidthLimit.TotalLimitGiB > 0 {
		remaining = append(remaining, server.BandwidthLimit.TotalLimitGiB*1024*1024-
			(server.BandwidthLimit.UplinkKiB-server.BandwidthLimit.UplinkInitialKiB)-
			(server.BandwidthLimit.DownlinkKiB-server.BandwidthLimit.DownlinkInitialKiB))
	}
	if server.BandwidthLimit.UplinkLimitGiB+server.BandwidthLimit.DownlinkLimitGiB > 0 {
		remaining = append(remaining,
			server.BandwidthLimit.UplinkLimitGiB*1024*1024-(server.BandwidthLimit.UplinkKiB-server.BandwidthLimit.UplinkInitialKiB)+
				server.BandwidthLimit.DownlinkLimitGiB*1024*1024-(server.BandwidthLimit.DownlinkKiB-server.BandwidthLimit.DownlinkInitialKiB))
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
	fRemainingGiB := float64(remaining[0]) / 1024 / 1024
	// do not show if there is adequate bandwidth
	if fRemainingGiB > 500 {
		return server.Name
	}
	fields := regexp.MustCompile(`^\[(.+)]\s*(.+)$`).FindStringSubmatch(server.Name)
	if len(fields) == 3 {
		// [100Mbps] Racknerd -> [100Mbps 472.7GiB] Racknerd
		return fmt.Sprintf("[%v %.1fGiB] %v", fields[1], fRemainingGiB, fields[2])
	}
	// Racknerd -> [472.7GiB] Racknerd
	return fmt.Sprintf("[%.1fGiB] %v", fRemainingGiB, server.Name)
}

func ServerIpTypes(servers []string) map[string]uint8 {
	servers = common.Deduplicate(servers)
	var dm []string
	var mu sync.Mutex
	var typ = make(map[string]uint8)
	for _, s := range servers {
		if ip := net.ParseIP(s); ip == nil {
			dm = append(dm, s)
		} else if ip.To4() != nil {
			typ[s] = 1
		} else {
			typ[s] = 2
		}
	}
	var wg sync.WaitGroup
	for _, d := range dm {
		wg.Add(1)
		go func(d string) {
			defer wg.Done()
			addrs, err := net.LookupHost(d)
			if err != nil {
				mu.Lock()
				typ[d] = 1 | 2
				mu.Unlock()
				return
			}
			for _, a := range addrs {
				mu.Lock()
				if net.ParseIP(a).To4() != nil {
					typ[d] |= 1
				} else {
					typ[d] |= 2
				}
				mu.Unlock()
			}
		}(d)
	}
	wg.Wait()
	return typ
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
		Password:   PasswordReserve,
		Method:     "chacha20-ietf-poly1305",
	})
	var relays []model.Server
	var svrs []model.Server
	for _, server := range servers {
		if server.FailureCount >= model.MaxFailureCount {
			// do not return disconnected server
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
	switch filter := c.Param("filter"); filter {
	case "4", "6":
		var servers []string
		for _, s := range sip008.Servers {
			servers = append(servers, s.Server)
		}
		typ := ServerIpTypes(servers)
		var filtered []model.SIP008Server
		for _, s := range sip008.Servers {
			if s.Password == PasswordReserve {
				filtered = append(filtered, s)
				continue
			}
			if filter == "4" && typ[s.Server]&1 == 1 {
				filtered = append(filtered, s)
			} else if filter == "6" && typ[s.Server]&2 == 2 {
				filtered = append(filtered, s)
			}
		}
		sip008.Servers = filtered
	}
	c.JSON(http.StatusOK, sip008)
}
