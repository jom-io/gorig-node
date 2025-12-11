package main

import (
	"github.com/jom-io/gorig/bootstrap"
	"gorig-node/gnnode"
)

import _ "gorig-node/gnnode"

func main() {
	gnnode.RegServer()
	bootstrap.StartUp()
}
