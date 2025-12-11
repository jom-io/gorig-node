package gnhttp

import (
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/jom-io/gorig/httpx"
	"github.com/jom-io/gorig/utils/errors"
	"github.com/jom-io/gorig/utils/sys"
	"net/http"
	"time"
)

var gHttpServer *http.Server

// NewEngine builds inbound HTTP engine for direct request simulation in tests.
func NewEngine() *gin.Engine {
	gEngine := gin.New()

	gEngine.Use(httpx.Recovery())
	gEngine.Use(Logger())
	gEngine.Use(httpx.CORS())
	gEngine.Use(gzip.Gzip(gzip.DefaultCompression))

	registerAllApisToRouter(gEngine)
	return gEngine
}

func Start(port string) error {
	if gHttpServer != nil {
		return nil
	}

	gHttpServer = &http.Server{
		Addr:              port,
		Handler:           NewEngine(),
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		err := gHttpServer.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			sys.Error(" * gorig-node invoke http server failed: ", err.Error())
			sys.Exit(errors.Sys(err.Error()))
			return
		}
	}()

	return nil
}
