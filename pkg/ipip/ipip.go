package ipip

import (
	"context"
	jsoniter "github.com/json-iterator/go"
	"net"
	"net/http"
	"sync"
)

var cache = make(map[string][]string)
var mu sync.Mutex

func IsChinaIPLookupTable(ip string) bool {
	ipObj := net.ParseIP(ip)
	if ipObj == nil {
		return false
	}
	ok, err := t.Contains(ipObj)
	return err == nil && ok
}

func GetLocation(ctx context.Context, ip string) []string {
	if net.ParseIP(ip) == nil {
		return nil
	}
	mu.Lock()
	defer mu.Unlock()
	if res, ok := cache[ip]; ok {
		return res
	}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://freeapi.ipip.net/"+ip, nil)
	if err != nil {
		return nil
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil
	}
	var loc []string
	if err := jsoniter.NewDecoder(resp.Body).Decode(&loc); err != nil {
		return nil
	}
	cache[ip] = loc
	return loc
}
