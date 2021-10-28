package model

import (
	"crypto/sha1"
	"fmt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/common"
	"reflect"
	"strings"
	"time"
)

const (
	BucketServer    = "server"
	MaxFailureCount = 10
)

type Protocol string

const (
	ProtocolVMessTCP    Protocol = "vmess"
	ProtocolShadowsocks          = "shadowsocks"
)

func (p Protocol) Valid() bool {
	switch p {
	case ProtocolVMessTCP, ProtocolShadowsocks:
		return true
	default:
		return false
	}
}

type Server struct {
	// Every server should have a server ticket, which should be included in each API interactions
	Ticket string
	// Name is also the proxy node name
	Name string
	// Hosts can be IPs and domains (split by ",")
	Hosts string `json:"Host"`
	// Port is shared by management and proxy
	Port int
	// Argument is used to connect and manage the server
	Argument Argument
	// BandwidthLimit is the limit of bandwidth
	BandwidthLimit BandwidthLimit

	// FailureCount is the number of consecutive failed pings
	FailureCount int
	// LastSeen is the time of last succeed ping
	LastSeen time.Time
	// SyncNextSeen is a flag indicates the server should be sync next seen
	SyncNextSeen bool
}

type BandwidthLimit struct {
	Valid bool
	// ResetDay is the day of every month to reset the limit of bandwidth. Zero means never reset.
	// This field should only be updated by SweetLisa after the first setup.
	ResetDay time.Time `json:",omitempty"`

	// UplinkLimitGiB is the limit of uplink bandwidth in GiB. Zero means no limit.
	UplinkLimitGiB uint64 `json:",omitempty"`
	// DownlinkLimitGiB is the limit of downlink bandwidth in GiB Zero means no limit.
	DownlinkLimitGiB uint64 `json:",omitempty"`
	// TotalLimitGiB is the limit of downlink plus uplink bandwidth in GiB Zero means no limit.
	TotalLimitGiB uint64 `json:",omitempty"`

	// UplinkKiB is the "transmit bytes" in /proc/net/dev of the biggest iface.
	UplinkKiB uint64 `json:",omitempty"`
	// DownlinkKiB is the "receive bytes" in /proc/net/dev of the biggest iface.
	DownlinkKiB uint64 `json:",omitempty"`

	// UplinkInitialKiB is the UplinkKiB at the beginning of the every cycles.
	UplinkInitialKiB uint64 `json:",omitempty"`
	// DownlinkInitialKiB is the DownlinkKiB at the beginning of the every cycles.
	DownlinkInitialKiB uint64 `json:",omitempty"`
}

func (l *BandwidthLimit) Exhausted() bool {
	if l.DownlinkLimitGiB != 0 && l.DownlinkKiB >= l.DownlinkInitialKiB+1024*1024*l.DownlinkLimitGiB {
		return true
	}
	if l.UplinkLimitGiB != 0 && l.UplinkKiB >= l.UplinkInitialKiB+1024*1024*l.UplinkLimitGiB {
		return true
	}
	if l.TotalLimitGiB != 0 && l.UplinkKiB+l.DownlinkKiB >= l.UplinkInitialKiB+l.DownlinkInitialKiB+1024*1024*l.TotalLimitGiB {
		return true
	}
	return false
}

func (l *BandwidthLimit) Update(r BandwidthLimit) {
	if !r.Valid {
		return
	}
	if !l.ResetDay.IsZero() && l.ResetDay.In(r.ResetDay.Location()).Day() == r.ResetDay.Day() {
		// update the statistic data
		l.DownlinkLimitGiB = r.DownlinkLimitGiB
		l.UplinkLimitGiB = r.UplinkLimitGiB
		l.TotalLimitGiB = r.TotalLimitGiB
		l.DownlinkKiB = r.DownlinkKiB
		l.UplinkKiB = r.UplinkKiB
	} else {
		// (re-)initiate
		now := time.Now()
		*l = BandwidthLimit{
			ResetDay: time.Date(now.Year(), now.Month(), r.ResetDay.Day(),
				0, 0, 0, 0, r.ResetDay.Location()).AddDate(0, 1, 0),
			UplinkLimitGiB:     r.UplinkLimitGiB,
			DownlinkLimitGiB:   r.DownlinkLimitGiB,
			TotalLimitGiB:      r.TotalLimitGiB,
			UplinkKiB:          r.UplinkKiB,
			DownlinkKiB:        r.DownlinkKiB,
			UplinkInitialKiB:   r.UplinkKiB,
			DownlinkInitialKiB: r.DownlinkKiB,
		}
	}
}

func (l *BandwidthLimit) IsTimeToReset() bool {
	if !l.ResetDay.IsZero() && time.Now().After(l.ResetDay) {
		return true
	}
	return false
}

func (l *BandwidthLimit) Reset() {
	l.UplinkInitialKiB = l.UplinkKiB
	l.DownlinkInitialKiB = l.DownlinkKiB
	l.ResetDay.AddDate(0, 1, 0)
}

func GetFirstHost(host string) string {
	return strings.SplitN(host, ",", 2)[0]
}

func GetUserArgument(serverTicket, userTicket string) Argument {
	h := sha1.New()
	h.Write([]byte(serverTicket))
	h.Write([]byte(userTicket))
	b := h.Sum(nil)
	return Argument{
		Protocol: ProtocolShadowsocks,
		Password: common.Base62Encoder.Encode(b)[:21],
		Method:   "chacha20-ietf-poly1305",
	}
}

func GetRelayUserArgument(serverTicket, relayTicket, userTicket string) Argument {
	h := sha1.New()
	h.Write([]byte(serverTicket))
	h.Write([]byte(relayTicket))
	h.Write([]byte(userTicket))
	b := h.Sum(nil)
	return Argument{
		Protocol: ProtocolShadowsocks,
		Password: common.Base62Encoder.Encode(b)[:21],
		Method:   "chacha20-ietf-poly1305",
	}
}

type Argument struct {
	// Required
	Protocol Protocol `json:",omitempty"`
	// Optional
	Username string `json:",omitempty"`
	// Required
	Password string `json:",omitempty"`
	// Optional
	Method string `json:",omitempty"`
}

func (a Argument) Hash() string {
	h := sha1.New()
	v := reflect.ValueOf(a)
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		h.Write([]byte(fmt.Sprint(field.Interface())))
	}
	return common.Base95Encoder.Encode(h.Sum(nil))
}
