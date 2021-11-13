package sharing_link

import (
	"encoding/base64"
	jsoniter "github.com/json-iterator/go"
	"strings"
)

type V2RayN struct {
	Ps            string `json:"ps"`
	Add           string `json:"add"`
	Port          string `json:"port"`
	ID            string `json:"id"`
	Aid           string `json:"aid"`
	Net           string `json:"net,omitempty"`
	Type          string `json:"type,omitempty"`
	Host          string `json:"host,omitempty"`
	Path          string `json:"path,omitempty"`
	TLS           string `json:"tls,omitempty"`
	Sni           string `json:"sni,omitempty"`
	V             string `json:"v"`
}

func (v *V2RayN) ExportToURL() string {
	v.V = "2"
	b, _ := jsoniter.Marshal(v)
	return "vmess://" + strings.TrimSuffix(base64.StdEncoding.EncodeToString(b), "=")
}
