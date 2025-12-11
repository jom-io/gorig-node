package gnhttp

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jom-io/gorig-node/client/register"
)

func registerAllApisToRouter(router *gin.Engine) {
	for name, srv := range register.RegisteredServers() {
		srv := srv
		for _, api := range srv.Apis {
			api := api
			path := fmt.Sprintf("/%s/%s", name, api.Method)
			router.POST(path, func(c *gin.Context) {
				handleAPIRequest(srv, api, c)
			})
		}
	}
}
