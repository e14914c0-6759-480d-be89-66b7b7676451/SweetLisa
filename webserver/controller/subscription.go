package controller

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/common"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/config"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/model"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/model/sharing_link"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/pkg/log"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/service"
	"github.com/gin-gonic/gin"
	"github.com/mzz2017/softwind/protocol"
	"github.com/rs/dnscache"
	"inet.af/netaddr"
	"net"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const PasswordReserve = "__SWEETLISA__"

var cachedResolver = dnscache.Resolver{}

func NameToShow(server *model.Server, showQuota bool, noQuota bool) string {
	remaining := make([]int64, 0, 3)
	if server.BandwidthLimit.TotalLimitGiB > 0 {
		remaining = append(remaining, server.BandwidthLimit.TotalLimitGiB*1024*1024-
			(server.BandwidthLimit.UplinkKiB-server.BandwidthLimit.UplinkInitialKiB)-
			(server.BandwidthLimit.DownlinkKiB-server.BandwidthLimit.DownlinkInitialKiB))
	}
	if server.BandwidthLimit.UplinkLimitGiB+server.BandwidthLimit.DownlinkLimitGiB > 0 {
		var r int64
		if server.BandwidthLimit.UplinkLimitGiB > 0 {
			r += server.BandwidthLimit.UplinkLimitGiB*1024*1024 - (server.BandwidthLimit.UplinkKiB - server.BandwidthLimit.UplinkInitialKiB)
		}
		if server.BandwidthLimit.DownlinkLimitGiB > 0 {
			r += server.BandwidthLimit.DownlinkLimitGiB*1024*1024 - (server.BandwidthLimit.DownlinkKiB - server.BandwidthLimit.DownlinkInitialKiB)
		}
		remaining = append(remaining, r)
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
	if noQuota || (fRemainingGiB > 500 && !showQuota) {
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

// GetSubscription returns the user's subscription
func GetSubscription(c *gin.Context) {
	ticket := c.Param("Ticket")
	ticObj := c.MustGet("Ticket").(*model.Ticket)
	switch ticObj.Type {
	case model.TicketTypeUser, model.TicketTypeRelay:
	default:
		ResponseError(c, fmt.Errorf("bad request"))
		return
	}
	// get servers
	servers, err := service.GetServersByChatIdentifier(nil, ticObj.ChatIdentifier, true)
	if err != nil {
		ResponseError(c, err)
		return
	}
	sort.Slice(servers, func(i, j int) bool {
		return servers[i].Name < servers[j].Name
	})

	// parse flags
	flags := strings.Split(c.Param("flags"), ",")
	var v4v6Mask uint8
	var typeMask uint8
	var showQuota bool
	var noQuota bool
	for _, flag := range flags {
		switch flag {
		case "4":
			v4v6Mask |= 1 << 0
		case "6":
			v4v6Mask |= 1 << 1
		case "quota":
			showQuota = true
		case "noquota":
			noQuota = true
		case "endpoint":
			typeMask |= 1 << 0
		case "relay":
			typeMask |= 1 << 1
		}
	}
	if v4v6Mask == 0 {
		v4v6Mask = 1 | 2
	}
	if typeMask == 0 {
		typeMask = 1 | 2
	}

	// generate sharing link
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
		switch svrTic.Type {
		case model.TicketTypeRelay:
			relays = append(relays, server)
		case model.TicketTypeServer:
			svrs = append(svrs, server)
		}
	}
	sort.Slice(svrs, func(i, j int) bool {
		return svrs[i].Name < svrs[j].Name
	})
	sort.Slice(relays, func(i, j int) bool {
		return relays[i].Name < relays[j].Name
	})

	maxCnt := 1 // alert node
	cnt := 1
	for _, svr := range svrs {
		maxCnt += len(strings.Split(svr.Hosts, ","))
	}
	for _, relay := range relays {
		for _, svr := range svrs {
			if svr.NoRelay {
				continue
			}
			maxCnt += len(strings.Split(relay.Hosts, ","))
		}
	}
	var mutex sync.Mutex
	lines := make([]string, maxCnt)

	alert := sharing_link.SIP002{
		Name:     fmt.Sprintf("ExpireAt: %v", ticObj.ExpireAt.Format("2006-01-02 15:04 MST")),
		Server:   "127.0.0.1",
		Port:     1024,
		Password: PasswordReserve,
		Cipher:   "chacha20-ietf-poly1305",
		Plugin:   sharing_link.SIP003{},
	}
	lines[0] = alert.ExportToURL()

	var wg sync.WaitGroup
	if (typeMask & 1) == 1 {
		for i, svr := range svrs {
			arg := model.GetUserArgument(svr.Ticket, ticket, svr.Argument.Protocol)
			hosts := strings.Split(svr.Hosts, ",")
			for _, host := range hosts {
				wg.Add(1)
				go func(cnt int, host string, svr *model.Server) {
					defer wg.Done()
					// it costs time
					if !ValidNetwork(host, v4v6Mask) {
						return
					}
					//log.Trace("svr: protocol: %v, host: %v, passwd: %v", arg.Protocol, host, arg.Password)
					switch arg.Protocol {
					case protocol.ProtocolShadowsocks:
						//log.Trace("shadowsocks")
						s := sharing_link.SIP002{
							Name:     NameToShow(svr, showQuota, noQuota),
							Server:   host,
							Port:     svr.Port,
							Password: arg.Password,
							Cipher:   arg.Method,
							Plugin:   sharing_link.SIP003{},
						}
						mutex.Lock()
						lines[cnt] = s.ExportToURL()
						mutex.Unlock()
					case protocol.ProtocolVMessTCP:
						//log.Trace("vmess")
						s := sharing_link.V2RayN{
							Ps:   NameToShow(svr, showQuota, noQuota),
							Add:  host,
							Port: strconv.Itoa(svr.Port),
							ID:   arg.Password,
							Aid:  "0",
							Net:  "tcp",
							Type: "none",
							V:    "2",
						}
						mutex.Lock()
						lines[cnt] = s.ExportToURL()
						mutex.Unlock()
					case protocol.ProtocolVMessTlsGrpc:
						//log.Trace("vmess+tls+grpc")
						sni, _ := common.HostToSNI(hosts[0], config.GetConfig().Host)
						s := sharing_link.V2RayN{
							Ps:   NameToShow(svr, showQuota, noQuota),
							Add:  host,
							Port: strconv.Itoa(svr.Port),
							ID:   arg.Password,
							Aid:  "0",
							Net:  "grpc",
							TLS:  "tls",
							Type: "none",
							Sni:  sni,
							Host: sni,
							Path: common.SimplyGetParam(arg.Method, "serviceName"),
							V:    "2",
						}
						mutex.Lock()
						lines[cnt] = s.ExportToURL()
						mutex.Unlock()
					default:
						log.Warn("unexpected protocol: %v", arg.Protocol)
					}
				}(cnt, host, &svrs[i])
				cnt++
			}
		}
	}

	if (typeMask & 2) == 2 {
		for i, relay := range relays {
			for j, svr := range svrs {
				if svr.NoRelay {
					continue
				}
				arg := model.GetRelayUserArgument(svr.Ticket, relay.Ticket, ticket, relay.Argument.Protocol)
				hosts := strings.Split(relay.Hosts, ",")
				for _, host := range hosts {
					wg.Add(1)
					go func(cnt int, host string, relay *model.Server, svr *model.Server) {
						defer wg.Done()
						// it costs time
						if !ValidNetwork(host, v4v6Mask) {
							return
						}
						switch arg.Protocol {
						case protocol.ProtocolShadowsocks:
							s := sharing_link.SIP002{
								Name:     fmt.Sprintf("%v -> %v", NameToShow(relay, showQuota, noQuota), NameToShow(svr, showQuota, noQuota)),
								Server:   host,
								Port:     relay.Port,
								Password: arg.Password,
								Cipher:   arg.Method,
								Plugin:   sharing_link.SIP003{},
							}
							mutex.Lock()
							lines[cnt] = s.ExportToURL()
							mutex.Unlock()
						case protocol.ProtocolVMessTCP:
							s := sharing_link.V2RayN{
								Ps:   fmt.Sprintf("%v -> %v", NameToShow(relay, showQuota, noQuota), NameToShow(svr, showQuota, noQuota)),
								Add:  host,
								Port: strconv.Itoa(relay.Port),
								ID:   arg.Password,
								Aid:  "0",
								Net:  "tcp",
								Type: "none",
								V:    "2",
							}
							mutex.Lock()
							lines[cnt] = s.ExportToURL()
							mutex.Unlock()
						case protocol.ProtocolVMessTlsGrpc:
							//log.Trace("vmess+tls+grpc")
							sni, _ := common.HostToSNI(hosts[0], config.GetConfig().Host)
							s := sharing_link.V2RayN{
								Ps:   fmt.Sprintf("%v -> %v", NameToShow(relay, showQuota, noQuota), NameToShow(svr, showQuota, noQuota)),
								Add:  host,
								Port: strconv.Itoa(relay.Port),
								ID:   arg.Password,
								Aid:  "0",
								Net:  "grpc",
								TLS:  "tls",
								Type: "none",
								Sni:  sni,
								Host: sni,
								Path: common.SimplyGetParam(arg.Method, "serviceName"),
								V:    "2",
							}
							mutex.Lock()
							lines[cnt] = s.ExportToURL()
							mutex.Unlock()
						default:
							log.Warn("unexpected protocol: %v", arg.Protocol)
						}
					}(cnt, host, &relays[i], &svrs[j])
					cnt++
				}
			}
		}
	}
	wg.Wait()
	// remove empty lines
	var i int
	for j := 0; j < len(lines); j++ {
		if lines[j] == "" {
			continue
		}
		if i != j {
			lines[i] = lines[j]
		}
		i++
	}
	lines = lines[:i]
	c.String(http.StatusOK, base64.StdEncoding.EncodeToString([]byte(strings.Join(lines, "\n"))))
}

func ValidNetwork(server string, v4v6Mask uint8) (ok bool) {
	return ServerNetType(server)&v4v6Mask > 0
}

func ServerNetType(server string) (typ uint8) {
	if ip, err := netaddr.ParseIP(server); err != nil {
		addrs, err := cachedResolver.LookupHost(context.Background(), server)
		if err != nil {
			// cannot resolve
			return 1 | 2
		}
		for _, a := range addrs {
			if net.ParseIP(a).To4() != nil {
				typ |= 1
			} else {
				typ |= 2
			}
		}
	} else if ip.IsLoopback() {
		typ = 1 | 2
	} else if ip.Is4() {
		typ = 1
	} else {
		typ = 2
	}
	return typ
}

func ResponseError(c *gin.Context, err error) {
	c.JSON(http.StatusOK, sharing_link.SIP008{
		Version: 1,
		Servers: []sharing_link.SIP008Server{{
			Remarks:    fmt.Sprintf("ERROR: %v", err),
			Server:     "127.0.0.1",
			ServerPort: 1024,
			Password:   PasswordReserve,
			Method:     "chacha20-ietf-poly1305",
			Plugin:     "",
			PluginOpts: "",
		}},
	})
}
