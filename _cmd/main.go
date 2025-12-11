package main

import (
	"github.com/jom-io/gorig-node/gnnode"
	"github.com/jom-io/gorig/bootstrap"
)

import _ "github.com/jom-io/gorig-node/gnnode"

func main() {
	gnnode.RegServer()
	bootstrap.StartUp()
}
