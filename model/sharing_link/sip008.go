package sharing_link

import jsoniter "github.com/json-iterator/go"

type SIP008 struct {
	Version        int            `json:"version"`
	Servers        []SIP008Server `json:"servers"`
	BytesUsed      int64          `json:"bytes_used"`
	BytesRemaining int64          `json:"bytes_remaining"`
}

type SIP008Server struct {
	Id         string `json:"id"`
	Remarks    string `json:"remarks"`
	Server     string `json:"server"`
	ServerPort int    `json:"server_port"`
	Password   string `json:"password"`
	Method     string `json:"method"`
	Plugin     string `json:"plugin"`
	PluginOpts string `json:"plugin_opts"`
}

func (s SIP008) ExportToString() string {
	b, _ := jsoniter.Marshal(s)
	return string(b)
}
