package cloudflare

import (
	"context"
	"fmt"
	"github.com/cloudflare/cloudflare-go"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/pkg/nameserver"
	"net/netip"
	"strings"
	"time"
)

func init() {
	nameserver.Register("cloudflare", New)
}

func New(token string) (nameserver.Nameserver, error) {
	api, err := cloudflare.NewWithAPIToken(token)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.TODO(), 15*time.Second)
	defer cancel()
	v, err := api.VerifyAPIToken(ctx)
	if err != nil {
		return nil, err
	}
	if v.Status != "active" {
		return nil, fmt.Errorf("invalid token")
	}
	return &Cloudflare{api: api}, nil
}

type Cloudflare struct {
	api *cloudflare.API
}

func (c *Cloudflare) Assign(ctx context.Context, domain string, strIP string) error {
	fields := strings.Split(domain, ".")
	if len(fields) < 2 {
		return fmt.Errorf("invalid domain: %v", domain)
	}
	ip, err := netip.ParseAddr(strIP)
	if err != nil {
		return err
	}
	var typ string
	if ip.Is6() {
		typ = "AAAA"
	} else {
		typ = "A"
	}
	zoneName := strings.Join(fields[len(fields)-2:], ".")
	zoneID, err := c.api.ZoneIDByName(zoneName)
	if err != nil {
		return err
	}
	records, err := c.api.DNSRecords(ctx, zoneID, cloudflare.DNSRecord{Name: domain, Type: typ})
	if err != nil {
		return err
	}
	f := false
	newRecord := cloudflare.DNSRecord{
		Type:    typ,
		Name:    domain,
		Content: strIP,
		TTL:     1, // 1 for 'automatic'
		Proxied: &f,
	}
	if len(records) > 0 {
		if records[0].Content == strIP && *records[0].Proxied == false {
			// no need to update
			return nil
		}
		return c.api.UpdateDNSRecord(ctx, zoneID, records[0].ID, newRecord)
	} else {
		_, err = c.api.CreateDNSRecord(ctx, zoneID, newRecord)
		return err
	}
}
