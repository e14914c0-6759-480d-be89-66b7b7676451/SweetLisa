package sharing_link

import (
	"net"
	"net/url"
	"strconv"

	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/common"
)

type Juicity struct {
	Name                  string
	Server                string
	Port                  int
	User                  string
	Password              string
	Sni                   string
	AllowInsecure         bool
	CongestionControl     string
	PinnedCertchainSha256 string
	Protocol              string
}

func (t *Juicity) ExportToURL() string {
	u := &url.URL{
		Scheme:   "juicity",
		User:     url.UserPassword(t.User, t.Password),
		Host:     net.JoinHostPort(t.Server, strconv.Itoa(t.Port)),
		Fragment: t.Name,
	}
	q := u.Query()
	if t.AllowInsecure {
		q.Set("allow_insecure", "1")
	}
	common.SetValue(&q, "sni", t.Sni)
	common.SetValue(&q, "congestion_control", t.CongestionControl)
	common.SetValue(&q, "pinned_certchain_sha256", t.PinnedCertchainSha256)
	u.RawQuery = q.Encode()
	return u.String()
}
