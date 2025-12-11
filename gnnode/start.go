package gnnode

import (
	"context"
	"fmt"
	"github.com/jom-io/gorig-node/client/inbound"
	"github.com/jom-io/gorig-node/client/register"
	"github.com/jom-io/gorig-node/gncfg"
	"github.com/jom-io/gorig/serv"
	"github.com/jom-io/gorig/utils/logger"
	"github.com/jom-io/gorig/utils/sys"
	"go.uber.org/zap"
	"time"
)

func RegServer() {
	nodeAddr := gncfg.Cfg.NodeAddr
	port := gncfg.DefNodePort
	if idx := len(nodeAddr) - 1; idx >= 0 {
		for i := idx; i >= 0; i-- {
			if nodeAddr[i] == ':' {
				port = nodeAddr[i:]
				break
			}
		}
	}
	err := serv.RegisterService(
		serv.Service{
			Code:     "GORIG-NODE",
			PORT:     port,
			Startup:  Startup,
			Shutdown: Shutdown,
		},
	)
	if err != nil {
		sys.Exit(err)
	}
}

func Startup(code, port string) error {
	sys.Info(fmt.Sprintf("  * %s service startup on port %s", code, port))
	go func() {
		// Delay slightly before registering to ensure service is ready.
		time.Sleep(200 * time.Millisecond)
		if err := register.Start(); err != nil {
			logger.Error(context.Background(), "  * register service start failed: ", zap.Error(err))
		}
		if err := inbound.StartInbound(port); err != nil {
			logger.Error(context.Background(), "  * inbound service start failed: ", zap.Error(err))
		}
	}()
	return nil
}

func Shutdown(code string, ctx context.Context) error {
	sys.Info("  * ", code, " service shutdown")
	register.Stop()
	if err := inbound.StopInbound(); err != nil {
		return err
	}
	return nil
}
