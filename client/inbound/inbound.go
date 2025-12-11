package inbound

import (
	"github.com/jom-io/gorig-node/client/inbound/gnhttp"
)

func StartInbound(addr string) error {
	return gnhttp.Start(addr)
}

func StopInbound() error {
	return nil
}
