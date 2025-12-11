package gnhttp

import (
	"github.com/gin-gonic/gin"
	"gorig-node/client/register"
	"io"
	"reflect"
)

func handleAPIRequest(srv *register.ServerRegister, api register.ApiInfo, c *gin.Context) {
	// 1. Locate handler
	fn, ok := srv.FnMap[api.Method]
	if !ok {
		c.JSON(404, gin.H{"error": "method not found"})
		return
	}

	meta := srv.MethodMeta[api.Method]

	// 2. Read body (wrapped request)
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// 3. Unpack arguments
	args, err := register.UnpackRequest(meta, body, reflect.ValueOf(c))
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// 4. Call original function
	results := fn.Call(args)

	// 5. Pack response
	respBytes, err := register.PackResponse(meta, results)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.Set(responseLogKey, string(respBytes))

	// 6. Always return 200; business errors stay in payload
	c.Data(200, "application/json", respBytes)
}
