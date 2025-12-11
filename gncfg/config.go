package gncfg

import (
	configure "github.com/jom-io/gorig/utils/cofigure"
)

const (
	DefNodePort = ":5807"
)

type GlobalConfig struct {
	HubAddr  string
	NodeAddr string
}

var Cfg GlobalConfig

func UseConfig(cfg GlobalConfig) {
	Cfg = cfg
}

func init() {
	hub := configure.GetString("gn.hub.addr", "")
	node := configure.GetString("gn.node.addr", "")
	Cfg = GlobalConfig{
		HubAddr:  hub,
		NodeAddr: node,
	}
}
